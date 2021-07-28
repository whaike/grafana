package notifiers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"golang.org/x/net/context"
	"golang.org/x/net/context/ctxhttp"
	"gopkg.in/yaml.v3"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/grafana/grafana/pkg/bus"
	"github.com/grafana/grafana/pkg/components/simplejson"
	"github.com/grafana/grafana/pkg/infra/log"
	m "github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/services/alerting"
)

type WeChatToken struct {
	AccessToken string `json:"access_token"`
	ErrMsg      string `json:"errmsg"`
	ErrCode     int    `json:"errcode"`
	ExpireIn    int    `json:"expire_in"`
}

type WeChatMediaId struct {
	MediaId string `json:"media_id"`
	ErrMsg  string `json:"errmsg"`
	ErrCode int    `json:"errcode"`
	CreatAt string `json:"created_at"`
	Type    string `json:"type"`
}

type Msg struct {
	Message     string    `json:"Message"`
	Severity    string    `json:"告警级别"`
	Alert       string    `json:"告警类型"`
	Summary     string    `json:"告警应用"`
	Host        []string  `json:"告警主机"`
	Description string    `json:"告警详情"`
	Expr        string    `json:"触发阀值"`
	CreateTime  time.Time `json:"告警时间"`
	Error       string    `json:"Error"`
	ImageUrl    string    `json:"ImageUrl"`
}

const (
	SEND_MESSAGE_ENDPOINT string = "https://qyapi.weixin.qq.com/cgi-bin/message/send"
	UPLOAD_IMAGE_ENDPOINT string = "https://qyapi.weixin.qq.com/cgi-bin/media/upload"
	GET_TOKEN_ENDPOINT    string = "https://qyapi.weixin.qq.com/cgi-bin/gettoken"
)

type AlterRuleTemplate struct {
	Template map[string]map[string]string `yaml:"global"`
}

var Temp *AlterRuleTemplate

func init() {
	y := new(AlterRuleTemplate)

	yamlFile, err := ioutil.ReadFile("/etc/grafana/wechat_alert_format.yaml")
	//yamlFile, err := ioutil.ReadFile("E:/grafana/pkg/services/alerting/notifiers/alert_rule.yaml")
	if err != nil {
		log.Error("read alert_rule.yaml fail", err.Error())
	}

	err = yaml.Unmarshal(yamlFile, y)
	if err != nil {
		log.Error("read alert_rule.yaml fail", err.Error())
	} else {
		Temp = y
	}
	alerting.RegisterNotifier(&alerting.NotifierPlugin{
		Type:        "wechat",
		Name:        "WeChat",
		Description: "Sends HTTP POST request to WeChat",
		Factory:     NewWeChatNotifier,
		Options: []alerting.NotifierOption{
			{
				Label:        "CorpId",
				Element:      alerting.ElementTypeInput,
				InputType:    alerting.InputTypeText,
				Placeholder:  "登陆察看“我的企业”，获取CorpID",
				PropertyName: "corpid",
				Required:     true,
			},
			{
				Label:        "AgentId",
				Element:      alerting.ElementTypeInput,
				InputType:    alerting.InputTypeText,
				Placeholder:  "打开告警应用，查看AgentId",
				PropertyName: "agentid",
				Required:     true,
			},
			{
				Label:        "Secret",
				Element:      alerting.ElementTypeInput,
				InputType:    alerting.InputTypeText,
				Placeholder:  "打开告警应用，查看Secret",
				PropertyName: "secret",
				Required:     true,
			},
			{
				Label:        "ToUser",
				Element:      alerting.ElementTypeInput,
				InputType:    alerting.InputTypeText,
				Placeholder:  "成员ID列表,多个接收者用‘|’分隔，最多支持1000个,指定为@all，则向关注该应用的全员发送",
				PropertyName: "touser",
				Required:     true,
			},
		},
	})
}

func NewWeChatNotifier(model *m.AlertNotification) (alerting.Notifier, error) {
	agentid := model.Settings.Get("agentid").MustString()
	if agentid == "" {
		return nil, alerting.ValidationError{Reason: "Could not find agentid property in settings"}
	}

	corpid := model.Settings.Get("corpid").MustString()
	if corpid == "" {
		return nil, alerting.ValidationError{Reason: "Could not find corpid property in settings"}
	}

	secret := model.Settings.Get("secret").MustString()
	if secret == "" {
		return nil, alerting.ValidationError{Reason: "Could not find secret property in settings"}
	}

	touser := model.Settings.Get("touser").MustString()
	if touser == "" {
		return nil, alerting.ValidationError{Reason: "Could not find touser property in settings"}
	}

	return &WeChatNotifier{
		NotifierBase: NewNotifierBase(model),
		AgentId:      agentid,
		CorpId:       corpid,
		Secret:       secret,
		ToUser:       touser,
		log:          log.New("alerting.notifier.wechat"),
	}, nil
}

type WeChatNotifier struct {
	NotifierBase
	AgentId string
	CorpId  string
	Secret  string
	ToUser  string
	log     log.Logger
}

func (this *WeChatNotifier) GetMediaId(path, token string) (string, error) {
	var mediaId string

	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	f, err := os.Open(path)
	if err != nil {
		return mediaId, err
	}
	defer f.Close()

	fw, err := w.CreateFormFile("media", path)
	if err != nil {
		return mediaId, err
	}

	_, err = io.Copy(fw, f)
	if err != nil {
		return mediaId, err
	}
	w.Close()

	url := fmt.Sprintf(UPLOAD_IMAGE_ENDPOINT+"?access_token=%s&type=image", token)
	request, err := http.NewRequest(http.MethodPost, url, &b)
	if err != nil {
		return mediaId, err
	}

	request.Header.Add("Content-Type", w.FormDataContentType())
	request.Header.Add("User-Agent", "Grafana")

	resp, err := ctxhttp.Do(context.TODO(), http.DefaultClient, request)
	if err != nil {
		return mediaId, err
	}

	if resp.StatusCode/100 != 2 {
		return mediaId, fmt.Errorf("WeChat returned statuscode invalid status code: %v", resp.Status)
	}
	defer resp.Body.Close()

	var wechatMediaId WeChatMediaId
	err = json.NewDecoder(resp.Body).Decode(&wechatMediaId)
	if err != nil {
		return mediaId, err
	}

	if wechatMediaId.ErrCode != 0 {
		return mediaId, fmt.Errorf("WeChat returned errmsg: %s", wechatMediaId.ErrMsg)
	}

	return wechatMediaId.MediaId, nil
}

func (this *WeChatNotifier) GetAccessToken() (string, error) {
	var token string

	url := fmt.Sprintf(GET_TOKEN_ENDPOINT+"?corpid=%s&corpsecret=%s", this.CorpId, this.Secret)
	request, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return token, err
	}

	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("User-Agent", "Grafana")

	resp, err := ctxhttp.Do(context.TODO(), http.DefaultClient, request)
	if err != nil {
		return token, err
	}

	if resp.StatusCode/100 != 2 {
		return token, fmt.Errorf("WeChat returned statuscode invalid status code: %v", resp.Status)
	}
	defer resp.Body.Close()

	var wechatToken WeChatToken
	err = json.NewDecoder(resp.Body).Decode(&wechatToken)
	if err != nil {
		return token, err
	}

	if wechatToken.ErrCode != 0 {
		return token, fmt.Errorf("WeChat returned errmsg: %s", wechatToken.ErrMsg)
	}
	return wechatToken.AccessToken, nil
}

func (this *WeChatNotifier) PushImage(evalContext *alerting.EvalContext, token string) error {
	mediaId, err := this.GetMediaId(evalContext.ImageOnDiskPath, token)
	if err != nil {
		return err
	}

	bodyJSON, err := simplejson.NewJson([]byte(`{
        "touser": "` + this.ToUser + `",
        "msgtype" : "image",
        "agentid": "` + this.AgentId + `",
        "image" : {
                "media_id": "` + mediaId + `"
        }
    }`))

	if err != nil {
		this.log.Error("Failed to create Json data", "error", err, "wechat", this.Name)
		return err
	}

	body, _ := bodyJSON.MarshalJSON()

	url := fmt.Sprintf(SEND_MESSAGE_ENDPOINT+"?access_token=%s", token)
	cmd := &m.SendWebhookSync{
		Url:  url,
		Body: string(body),
	}

	if err := bus.DispatchCtx(evalContext.Ctx, cmd); err != nil {
		this.log.Error("Failed to send WeChat", "error", err, "wechat", this.Name)
		return err
	}

	return nil
}

// Notify sends the alert notification to wechat.
func (this *WeChatNotifier) Notify(evalContext *alerting.EvalContext) error {
	this.log.Info("Sending wechat")

	token, err := this.GetAccessToken()
	if err != nil {
		this.log.Error("Get AccessToken failed", err)
		return err
	}

	var msg Msg
	var sourceHost []string
	//var metric string
	r := regexp.MustCompile(`(25[0-5]|2[0-4]\d|[0-1]?\d?\d)(\.(25[0-5]|2[0-4]\d|[0-1]?\d?\d)){3}:\d+`)
	for _, evt := range evalContext.EvalMatches {
		if evt.Metric != "" {
			sourceHost = append(sourceHost, r.FindAllString(evt.Metric, 1)[0])
		}
	}
	ruleName := strings.SplitN(evalContext.Rule.Name, "=", 1)[0]
	ruleTemp := Temp.Template[ruleName]

	if ruleTemp != nil {

		msg.Message = evalContext.Rule.Message
		msg.Alert = evalContext.Rule.Name
		msg.CreateTime = time.Now()
		msg.Description = ruleTemp["description"]
		msg.Summary = ruleTemp["summary"]
		msg.Expr = ruleTemp["expr"]
		msg.Host = sourceHost
		msg.Severity = ruleTemp["severity"]
	}
	fmt.Println(msg)
	if evalContext.Error != nil {
		msg.Error = evalContext.Error.Error()
	}

	if evalContext.ImageOnDiskPath == "" && evalContext.ImagePublicURL != "" {
		msg.ImageUrl = evalContext.ImagePublicURL
	}
	//this.log.Error("print msg", msg)
	//bodyJSON, err := simplejson.NewJson([]byte(`{
	//	"touser": "` + this.ToUser + `",
	//	"msgtype" : "text",
	//	"agentid": "` + this.AgentId + `",
	//	"text" : {
	//			"content": "` + strings.ReplaceAll(content, `"`, `'`) + `"
	//	}
	//}`))

	//if err != nil {
	//	this.log.Error("Failed to create Json data", "error", err, "wechat", this.Name)
	//	return err
	//}
	//
	//body, _ := bodyJSON.MarshalJSON()
	msgM, _ := json.Marshal(msg)
	var out bytes.Buffer
	json.Indent(&out, msgM, "", "    ")
	body, _ := json.Marshal(map[string]interface{}{
		"touser":  this.ToUser,
		"msgtype": "text",
		"agentid": this.AgentId,
		"text":    map[string]string{"content": out.String()},
	})

	url := fmt.Sprintf(SEND_MESSAGE_ENDPOINT+"?access_token=%s", token)
	cmd := &m.SendWebhookSync{
		Url:  url,
		Body: string(body),
	}

	if err := bus.DispatchCtx(evalContext.Ctx, cmd); err != nil {
		this.log.Error("Failed to send WeChat", "error", err, "wechat", this.Name)
		return err
	}

	if evalContext.ImageOnDiskPath != "" {
		err := this.PushImage(evalContext, token)
		if err != nil {
			this.log.Error("Failed to Push Image", "error", err, "path", evalContext.ImageOnDiskPath)
		}
	}

	return nil
}
