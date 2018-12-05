package notifiers

import (
	"os"
	"strconv"
	"time"

	"fmt"

	"github.com/grafana/grafana/pkg/bus"
	"github.com/grafana/grafana/pkg/components/simplejson"
	"github.com/grafana/grafana/pkg/log"
	m "github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/services/alerting"
)

func init() {
	alerting.RegisterNotifier(&alerting.NotifierPlugin{
		Type:        "pagerduty",
		Name:        "PagerDuty",
		Description: "向PagerDuty发送通知",
		Factory:     NewPagerdutyNotifier,
		OptionsTemplate: `
      <h3 class="page-heading">PagerDuty 设置</h3>
      <div class="gf-form">
        <span class="gf-form-label width-14">Integration Key</span>
        <input type="text" required class="gf-form-input max-width-22" ng-model="ctrl.model.settings.integrationKey" placeholder="Pagerduty集成密钥"></input>
      </div>
      <div class="gf-form">
        <gf-form-switch
           class="gf-form"
           label="自动解决事故"
           label-class="width-14"
           checked="ctrl.model.settings.autoResolve"
           tooltip="警报恢复正常后，解决pagerduty中的事件。">
        </gf-form-switch>
      </div>
    `,
	})
}

var (
	pagerdutyEventApiUrl = "https://events.pagerduty.com/v2/enqueue"
)

func NewPagerdutyNotifier(model *m.AlertNotification) (alerting.Notifier, error) {
	autoResolve := model.Settings.Get("autoResolve").MustBool(false)
	key := model.Settings.Get("integrationKey").MustString()
	if key == "" {
		return nil, alerting.ValidationError{Reason: "无法在设置中找到集成键属性"}
	}

	return &PagerdutyNotifier{
		NotifierBase: NewNotifierBase(model),
		Key:          key,
		AutoResolve:  autoResolve,
		log:          log.New("alerting.notifier.pagerduty"),
	}, nil
}

type PagerdutyNotifier struct {
	NotifierBase
	Key         string
	AutoResolve bool
	log         log.Logger
}

func (this *PagerdutyNotifier) Notify(evalContext *alerting.EvalContext) error {

	if evalContext.Rule.State == m.AlertStateOK && !this.AutoResolve {
		this.log.Info("不向Pagerduty发送触发器", "state", evalContext.Rule.State, "auto resolve", this.AutoResolve)
		return nil
	}

	eventType := "trigger"
	if evalContext.Rule.State == m.AlertStateOK {
		eventType = "resolve"
	}
	customData := triggMetrString
	for _, evt := range evalContext.EvalMatches {
		customData = customData + fmt.Sprintf("%s: %v\n", evt.Metric, evt.Value)
	}

	this.log.Info("Notifying Pagerduty", "event_type", eventType)

	payloadJSON := simplejson.New()
	payloadJSON.Set("summary", evalContext.Rule.Name+" - "+evalContext.Rule.Message)
	if hostname, err := os.Hostname(); err == nil {
		payloadJSON.Set("source", hostname)
	}
	payloadJSON.Set("severity", "critical")
	payloadJSON.Set("timestamp", time.Now())
	payloadJSON.Set("component", "Grafana")
	payloadJSON.Set("custom_details", customData)

	bodyJSON := simplejson.New()
	bodyJSON.Set("routing_key", this.Key)
	bodyJSON.Set("event_action", eventType)
	bodyJSON.Set("dedup_key", "alertId-"+strconv.FormatInt(evalContext.Rule.Id, 10))
	bodyJSON.Set("payload", payloadJSON)

	ruleUrl, err := evalContext.GetRuleUrl()
	if err != nil {
		this.log.Error("获取规则链接失败", "error", err)
		return err
	}
	links := make([]interface{}, 1)
	linkJSON := simplejson.New()
	linkJSON.Set("href", ruleUrl)
	bodyJSON.Set("client_url", ruleUrl)
	bodyJSON.Set("client", "Grafana")
	links[0] = linkJSON
	bodyJSON.Set("links", links)

	if evalContext.ImagePublicUrl != "" {
		contexts := make([]interface{}, 1)
		imageJSON := simplejson.New()
		imageJSON.Set("src", evalContext.ImagePublicUrl)
		contexts[0] = imageJSON
		bodyJSON.Set("images", contexts)
	}

	body, _ := bodyJSON.MarshalJSON()

	cmd := &m.SendWebhookSync{
		Url:        pagerdutyEventApiUrl,
		Body:       string(body),
		HttpMethod: "POST",
		HttpHeader: map[string]string{
			"Content-Type": "application/json",
		},
	}

	if err := bus.DispatchCtx(evalContext.Ctx, cmd); err != nil {
		this.log.Error("无法向Pagerduty发送通知", "error", err, "body", string(body))
		return err
	}

	return nil
}
