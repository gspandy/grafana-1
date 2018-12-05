package notifiers

import (
	"strconv"
	"strings"

	"github.com/grafana/grafana/pkg/bus"
	"github.com/grafana/grafana/pkg/components/simplejson"
	"github.com/grafana/grafana/pkg/log"
	m "github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/services/alerting"
)

func init() {
	alerting.RegisterNotifier(&alerting.NotifierPlugin{
		Type:        "sensu",
		Name:        "Sensu",
		Description: "将HTTP POST请求发送到Sensu API",
		Factory:     NewSensuNotifier,
		OptionsTemplate: `
      <h3 class="page-heading">Sensu 设置</h3>只需验证工具提示时间是否具有毫秒级分辨率
      <div class="gf-form">
        <span class="gf-form-label width-10">Url</span>
				<input type="text" required class="gf-form-input max-width-26" ng-model="ctrl.model.settings.url" placeholder="http://sensu-api.local:4567/results"></input>
      </div>
      <div class="gf-form">
        <span class="gf-form-label width-10">资源</span>
        <input type="text" class="gf-form-input max-width-14" ng-model="ctrl.model.settings.source" bs-tooltip="'如果将使用空规则ID'" data-placement="right"></input>
      </div>
      <div class="gf-form">
        <span class="gf-form-label width-10">处理器</span>
        <input type="text" class="gf-form-input max-width-14" ng-model="ctrl.model.settings.handler" placeholder="默认"></input>
      </div>
      <div class="gf-form">
        <span class="gf-form-label width-10">用户名</span>
        <input type="text" class="gf-form-input max-width-14" ng-model="ctrl.model.settings.username"></input>
      </div>
      <div class="gf-form">
        <span class="gf-form-label width-10">密码</span>
        <input type="text" class="gf-form-input max-width-14" ng-model="ctrl.model.settings.password"></input>
      </div>
    `,
	})

}

func NewSensuNotifier(model *m.AlertNotification) (alerting.Notifier, error) {
	url := model.Settings.Get("url").MustString()
	if url == "" {
		return nil, alerting.ValidationError{Reason: "在设置中找不到url属性"}
	}

	return &SensuNotifier{
		NotifierBase: NewNotifierBase(model),
		Url:          url,
		User:         model.Settings.Get("username").MustString(),
		Source:       model.Settings.Get("source").MustString(),
		Password:     model.Settings.Get("password").MustString(),
		Handler:      model.Settings.Get("handler").MustString(),
		log:          log.New("alerting.notifier.sensu"),
	}, nil
}

type SensuNotifier struct {
	NotifierBase
	Url      string
	Source   string
	User     string
	Password string
	Handler  string
	log      log.Logger
}

func (this *SensuNotifier) Notify(evalContext *alerting.EvalContext) error {
	this.log.Info("发送 sensu 结果")

	bodyJSON := simplejson.New()
	bodyJSON.Set("ruleId", evalContext.Rule.Id)
	// Sensu alerts cannot have spaces in them
	bodyJSON.Set("name", strings.Replace(evalContext.Rule.Name, " ", "_", -1))
	// Sensu alerts require a source. We set it to the user-specified value (optional),
	// else we fallback and use the grafana ruleID.
	if this.Source != "" {
		bodyJSON.Set("source", this.Source)
	} else {
		bodyJSON.Set("source", "grafana_rule_"+strconv.FormatInt(evalContext.Rule.Id, 10))
	}
	// Finally, sensu expects an output
	// We set it to a default output
	bodyJSON.Set("output", "Grafana Metric Condition Met")
	bodyJSON.Set("evalMatches", evalContext.EvalMatches)

	if evalContext.Rule.State == "alerting" {
		bodyJSON.Set("status", 2)
	} else if evalContext.Rule.State == "no_data" {
		bodyJSON.Set("status", 1)
	} else {
		bodyJSON.Set("status", 0)
	}

	if this.Handler != "" {
		bodyJSON.Set("handler", this.Handler)
	}

	ruleUrl, err := evalContext.GetRuleUrl()
	if err == nil {
		bodyJSON.Set("ruleUrl", ruleUrl)
	}

	if evalContext.ImagePublicUrl != "" {
		bodyJSON.Set("imageUrl", evalContext.ImagePublicUrl)
	}

	if evalContext.Rule.Message != "" {
		bodyJSON.Set("output", evalContext.Rule.Message)
	}

	body, _ := bodyJSON.MarshalJSON()

	cmd := &m.SendWebhookSync{
		Url:        this.Url,
		User:       this.User,
		Password:   this.Password,
		Body:       string(body),
		HttpMethod: "POST",
	}

	if err := bus.DispatchCtx(evalContext.Ctx, cmd); err != nil {
		this.log.Error("无法发送sensu事件", "error", err, "sensu", this.Name)
		return err
	}

	return nil
}
