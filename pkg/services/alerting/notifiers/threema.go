package notifiers

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/grafana/grafana/pkg/bus"
	"github.com/grafana/grafana/pkg/log"
	m "github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/services/alerting"
)

var (
	threemaGwBaseURL = "https://msgapi.threema.ch/%s"
)

func init() {
	alerting.RegisterNotifier(&alerting.NotifierPlugin{
		Type:        "threema",
		Name:        "Threema 网关",
		Description: "使用Threema Gateway向Threema发送通知",
		Factory:     NewThreemaNotifier,
		OptionsTemplate: `
      <h3 class="page-heading">Threema 网关设置</h3>
      <p>
        可以为任何类型的Threema网关ID配置通知"Basic"。目前不支持端到端ID。
      </p>
      <p>
        可以在以下位置设置Threema网关ID
        <a href="https://gateway.threema.ch/" target="_blank" rel="noopener noreferrer">https://gateway.threema.ch/</a>.
      </p>
      <div class="gf-form">
        <span class="gf-form-label width-14">网关 ID</span>
        <input type="text" required maxlength="8" pattern="\*[0-9A-Z]{7}"
          class="gf-form-input max-width-14"
          ng-model="ctrl.model.settings.gateway_id"
          placeholder="*3MAGWID">
        </input>
        <info-popover mode="right-normal">
          您的8个字符的Threema网关ID（以*开头）
        </info-popover>
      </div>
      <div class="gf-form">
        <span class="gf-form-label width-14">Recipient ID</span>
        <input type="text" required maxlength="8" pattern="[0-9A-Z]{8}"
          class="gf-form-input max-width-14"
          ng-model="ctrl.model.settings.recipient_id"
          placeholder="YOUR3MID">
        </input>
        <info-popover mode="right-normal">
          应该接收警报的8个字符的Threema ID
        </info-popover>
      </div>
      <div class="gf-form">
        <span class="gf-form-label width-14">API Secret</span>
        <input type="text" required
          class="gf-form-input max-width-24"
          ng-model="ctrl.model.settings.api_secret">
        </input>
        <info-popover mode="right-normal">
          您的Threema Gateway API秘密
        </info-popover>
      </div>
    `,
	})

}

type ThreemaNotifier struct {
	NotifierBase
	GatewayID   string
	RecipientID string
	APISecret   string
	log         log.Logger
}

func NewThreemaNotifier(model *m.AlertNotification) (alerting.Notifier, error) {
	if model.Settings == nil {
		return nil, alerting.ValidationError{Reason: "没有提供设置"}
	}

	gatewayID := model.Settings.Get("gateway_id").MustString()
	recipientID := model.Settings.Get("recipient_id").MustString()
	apiSecret := model.Settings.Get("api_secret").MustString()

	// Validation
	if gatewayID == "" {
		return nil, alerting.ValidationError{Reason: "在设置中找不到Threema Gateway ID"}
	}
	if !strings.HasPrefix(gatewayID, "*") {
		return nil, alerting.ValidationError{Reason: "无效的Threema网关ID：必须以*开头"}
	}
	if len(gatewayID) != 8 {
		return nil, alerting.ValidationError{Reason: "无效的Threema网关ID：长度必须为8个字符"}
	}
	if recipientID == "" {
		return nil, alerting.ValidationError{Reason: "在设置中找不到Threema收件人ID"}
	}
	if len(recipientID) != 8 {
		return nil, alerting.ValidationError{Reason: "在设置中找不到Threema收件人ID"}
	}
	if apiSecret == "" {
		return nil, alerting.ValidationError{Reason: "在设置中找不到Threema API密码"}
	}

	return &ThreemaNotifier{
		NotifierBase: NewNotifierBase(model),
		GatewayID:    gatewayID,
		RecipientID:  recipientID,
		APISecret:    apiSecret,
		log:          log.New("alerting.notifier.threema"),
	}, nil
}

func (notifier *ThreemaNotifier) Notify(evalContext *alerting.EvalContext) error {
	notifier.log.Info("Sending alert notification from", "threema_id", notifier.GatewayID)
	notifier.log.Info("Sending alert notification to", "threema_id", notifier.RecipientID)

	// Set up basic API request data
	data := url.Values{}
	data.Set("from", notifier.GatewayID)
	data.Set("to", notifier.RecipientID)
	data.Set("secret", notifier.APISecret)

	// Determine emoji
	stateEmoji := ""
	switch evalContext.Rule.State {
	case m.AlertStateOK:
		stateEmoji = "\u2705 " // White Heavy Check Mark
	case m.AlertStateNoData:
		stateEmoji = "\u2753 " // Black Question Mark Ornament
	case m.AlertStateAlerting:
		stateEmoji = "\u26A0 " // Warning sign
	}

	// Build message
	message := fmt.Sprintf("%s%s\n\n*State:* %s\n*Message:* %s\n",
		stateEmoji, evalContext.GetNotificationTitle(),
		evalContext.Rule.Name, evalContext.Rule.Message)
	ruleURL, err := evalContext.GetRuleUrl()
	if err == nil {
		message = message + fmt.Sprintf("*URL:* %s\n", ruleURL)
	}
	if evalContext.ImagePublicUrl != "" {
		message = message + fmt.Sprintf("*Image:* %s\n", evalContext.ImagePublicUrl)
	}
	data.Set("text", message)

	// Prepare and send request
	url := fmt.Sprintf(threemaGwBaseURL, "send_simple")
	body := data.Encode()
	headers := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}
	cmd := &m.SendWebhookSync{
		Url:        url,
		Body:       body,
		HttpMethod: "POST",
		HttpHeader: headers,
	}
	if err := bus.DispatchCtx(evalContext.Ctx, cmd); err != nil {
		notifier.log.Error("Failed to send webhook", "error", err, "webhook", notifier.Name)
		return err
	}

	return nil
}
