package notifiers

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"github.com/grafana/grafana/pkg/bus"
	"github.com/grafana/grafana/pkg/log"
	m "github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/services/alerting"
	"github.com/grafana/grafana/pkg/setting"
)

func init() {
	alerting.RegisterNotifier(&alerting.NotifierPlugin{
		Type:        "slack",
		Name:        "Slack",
		Description: "通过Slack Webhooks向Slack发送通知",
		Factory:     NewSlackNotifier,
		OptionsTemplate: `
      <h3 class="page-heading">Slack settings</h3>
      <div class="gf-form max-width-30">
        <span class="gf-form-label width-6">Url</span>
        <input type="text" required class="gf-form-input max-width-30" ng-model="ctrl.model.settings.url" placeholder="Slack 传入的 webhook url"></input>
      </div>
      <div class="gf-form max-width-30">
        <span class="gf-form-label width-6">Recipient</span>
        <input type="text"
          class="gf-form-input max-width-30"
          ng-model="ctrl.model.settings.recipient"
          data-placement="right">
        </input>
        <info-popover mode="right-absolute">
          覆盖默认频道或用户，使用#channel-name或@username
        </info-popover>
      </div>
      <div class="gf-form max-width-30">
        <span class="gf-form-label width-6">用户名</span>
        <input type="text"
          class="gf-form-input max-width-30"
          ng-model="ctrl.model.settings.username"
          data-placement="right">
        </input>
        <info-popover mode="right-absolute">
          设置机器人消息的用户名
        </info-popover>
      </div>
      <div class="gf-form max-width-30">
        <span class="gf-form-label width-6">Icon emoji</span>
        <input type="text"
          class="gf-form-input max-width-30"
          ng-model="ctrl.model.settings.icon_emoji"
          data-placement="right">
        </input>
        <info-popover mode="right-absolute">
          提供表情符号以用作机器人消息的图标。覆盖图标URL
        </info-popover>
      </div>
      <div class="gf-form max-width-30">
        <span class="gf-form-label width-6">Icon URL</span>
        <input type="text"
          class="gf-form-input max-width-30"
          ng-model="ctrl.model.settings.icon_url"
          data-placement="right">
        </input>
        <info-popover mode="right-absolute">
          提供图像的URL以用作机器人消息的图标
        </info-popover>
      </div>
      <div class="gf-form max-width-30">
        <span class="gf-form-label width-6">Mention</span>
        <input type="text"
          class="gf-form-input max-width-30"
          ng-model="ctrl.model.settings.mention"
          data-placement="right">
        </input>
        <info-popover mode="right-absolute">
          在通道中通知时，使用@提及用户或组
        </info-popover>
      </div>
      <div class="gf-form max-width-30">
        <span class="gf-form-label width-6">Token</span>
        <input type="text"
          class="gf-form-input max-width-30"
          ng-model="ctrl.model.settings.token"
          data-placement="right">
        </input>
        <info-popover mode="right-absolute">
          提供机器人令牌以使用Slack file.upload API（以“xoxb”开头）。在收件人中指定#channel-name或@username以使其生效 
        </info-popover>
      </div>
    `,
	})

}

func NewSlackNotifier(model *m.AlertNotification) (alerting.Notifier, error) {
	url := model.Settings.Get("url").MustString()
	if url == "" {
		return nil, alerting.ValidationError{Reason: "在设置中找不到url属性"}
	}

	recipient := model.Settings.Get("recipient").MustString()
	username := model.Settings.Get("username").MustString()
	iconEmoji := model.Settings.Get("icon_emoji").MustString()
	iconUrl := model.Settings.Get("icon_url").MustString()
	mention := model.Settings.Get("mention").MustString()
	token := model.Settings.Get("token").MustString()
	uploadImage := model.Settings.Get("uploadImage").MustBool(true)

	return &SlackNotifier{
		NotifierBase: NewNotifierBase(model),
		Url:          url,
		Recipient:    recipient,
		Username:     username,
		IconEmoji:    iconEmoji,
		IconUrl:      iconUrl,
		Mention:      mention,
		Token:        token,
		Upload:       uploadImage,
		log:          log.New("alerting.notifier.slack"),
	}, nil
}

type SlackNotifier struct {
	NotifierBase
	Url       string
	Recipient string
	Username  string
	IconEmoji string
	IconUrl   string
	Mention   string
	Token     string
	Upload    bool
	log       log.Logger
}

func (this *SlackNotifier) Notify(evalContext *alerting.EvalContext) error {
	this.log.Info("执行 slack 通知", "ruleId", evalContext.Rule.Id, "notification", this.Name)

	ruleUrl, err := evalContext.GetRuleUrl()
	if err != nil {
		this.log.Error("获取规则链接失败", "error", err)
		return err
	}

	fields := make([]map[string]interface{}, 0)
	fieldLimitCount := 4
	for index, evt := range evalContext.EvalMatches {
		fields = append(fields, map[string]interface{}{
			"title": evt.Metric,
			"value": evt.Value,
			"short": true,
		})
		if index > fieldLimitCount {
			break
		}
	}

	if evalContext.Error != nil {
		fields = append(fields, map[string]interface{}{
			"title": "错误信息",
			"value": evalContext.Error.Error(),
			"short": false,
		})
	}

	message := this.Mention
	if evalContext.Rule.State != m.AlertStateOK { //don't add message when going back to alert state ok.
		message += " " + evalContext.Rule.Message
	}
	image_url := ""
	// default to file.upload API method if a token is provided
	if this.Token == "" {
		image_url = evalContext.ImagePublicUrl
	}

	body := map[string]interface{}{
		"attachments": []map[string]interface{}{
			{
				"fallback":    evalContext.GetNotificationTitle(),
				"color":       evalContext.GetStateModel().Color,
				"title":       evalContext.GetNotificationTitle(),
				"title_link":  ruleUrl,
				"text":        message,
				"fields":      fields,
				"image_url":   image_url,
				"footer":      "Grafana v" + setting.BuildVersion,
				"footer_icon": "https://grafana.com/assets/img/fav32.png",
				"ts":          time.Now().Unix(),
			},
		},
		"parse": "full", // to linkify urls, users and channels in alert message.
	}

	//recipient override
	if this.Recipient != "" {
		body["channel"] = this.Recipient
	}
	if this.Username != "" {
		body["username"] = this.Username
	}
	if this.IconEmoji != "" {
		body["icon_emoji"] = this.IconEmoji
	}
	if this.IconUrl != "" {
		body["icon_url"] = this.IconUrl
	}
	data, _ := json.Marshal(&body)
	cmd := &m.SendWebhookSync{Url: this.Url, Body: string(data)}
	if err := bus.DispatchCtx(evalContext.Ctx, cmd); err != nil {
		this.log.Error("无法发送 slack 通知", "error", err, "webhook", this.Name)
		return err
	}
	if this.Token != "" && this.UploadImage {
		err = SlackFileUpload(evalContext, this.log, "https://slack.com/api/files.upload", this.Recipient, this.Token)
		if err != nil {
			return err
		}
	}
	return nil
}

func SlackFileUpload(evalContext *alerting.EvalContext, log log.Logger, url string, recipient string, token string) error {
	if evalContext.ImageOnDiskPath == "" {
		evalContext.ImageOnDiskPath = filepath.Join(setting.HomePath, "public/img/mixed_styles.png")
	}
	log.Info("通过file.upload API上传到slack")
	headers, uploadBody, err := GenerateSlackBody(evalContext.ImageOnDiskPath, token, recipient)
	if err != nil {
		return err
	}
	cmd := &m.SendWebhookSync{Url: url, Body: uploadBody.String(), HttpHeader: headers, HttpMethod: "POST"}
	if err := bus.DispatchCtx(evalContext.Ctx, cmd); err != nil {
		log.Error("无法上传slack的图像", "error", err, "webhook", "file.upload")
		return err
	}
	if err != nil {
		return err
	}
	return nil
}

func GenerateSlackBody(file string, token string, recipient string) (map[string]string, bytes.Buffer, error) {
	// Slack requires all POSTs to files.upload to present
	// an "application/x-www-form-urlencoded" encoded querystring
	// See https://api.slack.com/methods/files.upload
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	// Add the generated image file
	f, err := os.Open(file)
	if err != nil {
		return nil, b, err
	}
	defer f.Close()
	fw, err := w.CreateFormFile("file", file)
	if err != nil {
		return nil, b, err
	}
	_, err = io.Copy(fw, f)
	if err != nil {
		return nil, b, err
	}
	// Add the authorization token
	err = w.WriteField("token", token)
	if err != nil {
		return nil, b, err
	}
	// Add the channel(s) to POST to
	err = w.WriteField("channels", recipient)
	if err != nil {
		return nil, b, err
	}
	w.Close()
	headers := map[string]string{
		"Content-Type":  w.FormDataContentType(),
		"Authorization": "auth_token=\"" + token + "\"",
	}
	return headers, b, nil
}
