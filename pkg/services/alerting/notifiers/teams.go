package notifiers

import (
	"encoding/json"

	"github.com/grafana/grafana/pkg/bus"
	"github.com/grafana/grafana/pkg/log"
	m "github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/services/alerting"
)

func init() {
	alerting.RegisterNotifier(&alerting.NotifierPlugin{
		Type:        "teams",
		Name:        "微软团队",
		Description: "使用Incoming Webhook连接器向Microsoft Teams发送通知",
		Factory:     NewTeamsNotifier,
		OptionsTemplate: `
      <h3 class="page-heading">Teams 设置</h3>
      <div class="gf-form max-width-30">
        <span class="gf-form-label width-6">Url</span>
        <input type="text" required class="gf-form-input max-width-30" ng-model="ctrl.model.settings.url" placeholder="Teams incoming webhook url"></input>
      </div>
    `,
	})

}

func NewTeamsNotifier(model *m.AlertNotification) (alerting.Notifier, error) {
	url := model.Settings.Get("url").MustString()
	if url == "" {
		return nil, alerting.ValidationError{Reason: "在设置中找不到url属性"}
	}

	return &TeamsNotifier{
		NotifierBase: NewNotifierBase(model),
		Url:          url,
		log:          log.New("alerting.notifier.teams"),
	}, nil
}

type TeamsNotifier struct {
	NotifierBase
	Url string
	log log.Logger
}

func (this *TeamsNotifier) Notify(evalContext *alerting.EvalContext) error {
	this.log.Info("执行团队通知", "ruleId", evalContext.Rule.Id, "notification", this.Name)

	ruleUrl, err := evalContext.GetRuleUrl()
	if err != nil {
		this.log.Error("获取规则链接失败", "error", err)
		return err
	}

	fields := make([]map[string]interface{}, 0)
	fieldLimitCount := 4
	for index, evt := range evalContext.EvalMatches {
		fields = append(fields, map[string]interface{}{
			"name":  evt.Metric,
			"value": evt.Value,
		})
		if index > fieldLimitCount {
			break
		}
	}

	if evalContext.Error != nil {
		fields = append(fields, map[string]interface{}{
			"name":  "错误信息",
			"value": evalContext.Error.Error(),
		})
	}

	message := ""
	if evalContext.Rule.State != m.AlertStateOK { //don't add message when going back to alert state ok.
		message = evalContext.Rule.Message
	}

	body := map[string]interface{}{
		"@type":    "MessageCard",
		"@context": "http://schema.org/extensions",
		// summary MUST not be empty or the webhook request fails
		// summary SHOULD contain some meaningful information, since it is used for mobile notifications
		"summary":    evalContext.GetNotificationTitle(),
		"title":      evalContext.GetNotificationTitle(),
		"themeColor": evalContext.GetStateModel().Color,
		"sections": []map[string]interface{}{
			{
				"title": "Details",
				"facts": fields,
				"images": []map[string]interface{}{
					{
						"image": evalContext.ImagePublicUrl,
					},
				},
				"text": message,
			},
		},
		"potentialAction": []map[string]interface{}{
			{
				"@context": "http://schema.org",
				"@type":    "OpenUri",
				"name":     "规则视图",
				"targets": []map[string]interface{}{
					{
						"os": "default", "uri": ruleUrl,
					},
				},
			},
			{
				"@context": "http://schema.org",
				"@type":    "OpenUri",
				"name":     "图标视图",
				"targets": []map[string]interface{}{
					{
						"os": "default", "uri": evalContext.ImagePublicUrl,
					},
				},
			},
		},
	}

	data, _ := json.Marshal(&body)
	cmd := &m.SendWebhookSync{Url: this.Url, Body: string(data)}

	if err := bus.DispatchCtx(evalContext.Ctx, cmd); err != nil {
		this.log.Error("无法发送团队通知>", "error", err, "webhook", this.Name)
		return err
	}

	return nil
}
