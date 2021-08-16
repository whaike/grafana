package notifiers

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/grafana/grafana/pkg/bus"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/services/alerting"
)

const defaultDingdingMsgType = "link"

func init() {
	alerting.RegisterNotifier(&alerting.NotifierPlugin{
		Type:        "dingding",
		Name:        "DingDing",
		Description: "Sends HTTP POST request to DingDing",
		Heading:     "DingDing settings",
		Factory:     newDingDingNotifier,
		Options: []alerting.NotifierOption{
			{
				Label:        "Url",
				Element:      alerting.ElementTypeInput,
				InputType:    alerting.InputTypeText,
				Placeholder:  "https://oapi.dingtalk.com/robot/send?access_token=xxxxxxxxx",
				PropertyName: "url",
				Required:     true,
			},
			{
				Label:        "Message Type",
				Element:      alerting.ElementTypeSelect,
				PropertyName: "msgType",
				SelectOptions: []alerting.SelectOption{
					{
						Value: "link",
						Label: "Link",
					},
					{
						Value: "actionCard",
						Label: "ActionCard",
					},
					{
						Value: "text",
						Label: "Text",
					},
				},
			},
			{
				Label:        "At",
				Element:      alerting.ElementTypeInput,
				InputType:    alerting.InputTypeText,
				Placeholder:  "18989898889,多个号码用逗号隔开,@所有人用 @all ,仅支持 Text 消息类型",
				PropertyName: "msgAt",
				Required:     true,
			},
		},
	})
}

func newDingDingNotifier(model *models.AlertNotification) (alerting.Notifier, error) {
	url := model.Settings.Get("url").MustString()
	if url == "" {
		return nil, alerting.ValidationError{Reason: "Could not find url property in settings"}
	}

	msgType := model.Settings.Get("msgType").MustString(defaultDingdingMsgType)

	msgAt := model.Settings.Get("msgAt").MustString()

	return &DingDingNotifier{
		NotifierBase: NewNotifierBase(model),
		MsgType:      msgType,
		URL:          url,
		MsgAt:        msgAt,
		log:          log.New("alerting.notifier.dingding"),
	}, nil
}

// DingDingNotifier is responsible for sending alert notifications to ding ding.
type DingDingNotifier struct {
	NotifierBase
	MsgType string
	URL     string
	MsgAt   string
	log     log.Logger
}

// Notify sends the alert notification to dingding.
func (dd *DingDingNotifier) Notify(evalContext *alerting.EvalContext) error {
	dd.log.Info("Sending dingding")
	dd.log.Info(dd.MsgType)
	dd.log.Info(dd.MsgAt)
	messageURL, err := evalContext.GetRuleURL()
	if err != nil {
		dd.log.Error("Failed to get messageUrl", "error", err, "dingding", dd.Name)
		messageURL = ""
	}

	body, err := dd.genBody(evalContext, messageURL)
	if err != nil {
		return err
	}

	dd.log.Info(string(body))

	cmd := &models.SendWebhookSync{
		Url:  dd.URL,
		Body: string(body),
	}

	if err := bus.DispatchCtx(evalContext.Ctx, cmd); err != nil {
		dd.log.Error("Failed to send DingDing", "error", err, "dingding", dd.Name)
		return err
	}

	return nil
}

func (dd *DingDingNotifier) genBody(evalContext *alerting.EvalContext, messageURL string) ([]byte, error) {
	q := url.Values{
		"pc_slide": {"false"},
		"url":      {messageURL},
	}

	// Use special link to auto open the message url outside of Dingding
	// Refer: https://open-doc.dingtalk.com/docs/doc.htm?treeId=385&articleId=104972&docType=1#s9
	messageURL = "dingtalk://dingtalkclient/page/link?" + q.Encode()

	dd.log.Info("messageUrl:" + messageURL)

	message := evalContext.Rule.Message
	picURL := evalContext.ImagePublicURL
	title := evalContext.GetNotificationTitle()
	if message == "" {
		message = title
	}

	for i, match := range evalContext.EvalMatches {
		message += fmt.Sprintf("\n%2d. %s: %s", i+1, match.Metric, match.Value)
	}
	dd.log.Info(dd.MsgType)

	var bodyMsg map[string]interface{}
	if dd.MsgType == "actionCard" {
		// Embed the pic into the markdown directly because actionCard doesn't have a picUrl field
		if dd.NeedsImage() && picURL != "" {
			message = "![](" + picURL + ")\n\n" + message
		}

		bodyMsg = map[string]interface{}{
			"msgtype": "actionCard",
			"actionCard": map[string]interface{}{
				"text":        message,
				"title":       title,
				"singleTitle": "More",
				"singleURL":   messageURL,
			},
		}
	} else if dd.MsgType == "text" {
		at := map[string]interface{}{}
		msgAt := dd.MsgAt
		if strings.HasPrefix(msgAt, "@") && msgAt == "@all" {
			at = map[string]interface{}{
				"isAtAll": true,
			}
		} else if strings.Contains(msgAt, ",") {
			tmps := strings.Split(msgAt, ",")
			tmp := make([]string, 0)
			for _, t := range tmps {
				tmp = append(tmp, t)
			}
			at = map[string]interface{}{
				"atMobiles": tmp,
			}
		} else if len(msgAt) == 11 {
			tmp := make([]string, 0)
			tmp = append(tmp, msgAt)
			at = map[string]interface{}{
				"atMobiles": tmp,
			}
		} else {
			at = map[string]interface{}{
				"isAtAll": false,
			}
		}

		bodyMsg = map[string]interface{}{
			"at": at,
			"text": map[string]string{
				"content": "alert:" + message},
			"msgtype": "text",
		}

	} else {
		link := map[string]interface{}{
			"text":       message,
			"title":      title,
			"messageUrl": messageURL,
		}

		if dd.NeedsImage() {
			link["picUrl"] = picURL
		}

		bodyMsg = map[string]interface{}{
			"msgtype": "link",
			"link":    link,
		}
	}
	return json.Marshal(bodyMsg)
}
