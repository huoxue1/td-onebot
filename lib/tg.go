package lib

import (
	"bufio"
	"errors"
	"fmt"
	qrcodeTerminal "github.com/Baozisoftware/qrcode-terminal-go"
	"github.com/botuniverse/go-libonebot"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/telegram/auth/qrlogin"
	"github.com/gotd/td/telegram/dcs"
	"github.com/gotd/td/telegram/updates"
	"github.com/gotd/td/telegram/updates/hook"
	"github.com/gotd/td/tg"
	log "github.com/sirupsen/logrus"
	"github.com/skip2/go-qrcode"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/net/context"
	"golang.org/x/net/proxy"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"td-onebot/conf"
	"time"
)

var (
	dispatcher tg.UpdateDispatcher
)

func Init() {

	var l *zap.Logger
	if log.GetLevel() == log.DebugLevel {
		l, _ = zap.NewDevelopment(zap.IncreaseLevel(zapcore.DebugLevel), zap.AddStacktrace(zapcore.FatalLevel))
	} else {
		l, _ = zap.NewProduction(zap.IncreaseLevel(zapcore.InfoLevel), zap.AddStacktrace(zapcore.FatalLevel))
	}

	dispatcher = tg.NewUpdateDispatcher()
	gaps := updates.New(updates.Config{
		Handler: dispatcher,
		Logger:  l.Named("gaps"),
	})
	InitTg(context.Background(), gaps)
}

func onConnect(ctx context.Context, client *telegram.Client) {
	user, _ := client.Self(ctx)
	bot := &Bot{
		Self: user,
		Ob: libonebot.NewOneBot("td-onebot", &libonebot.Self{
			Platform: "td",
			UserID:   strconv.FormatInt(user.ID, 10),
		}, &conf.GetConfig().Config),
		Client: client,
		ctx:    ctx,
		config: make(map[string]any, 10),
	}
	bot.Ob.Logger = log.StandardLogger()

	handleEvent(bot)
	handleApi(bot)
	go bot.Ob.Run()
}

func InitTg(ctx context.Context, manager *updates.Manager) {
	config := conf.GetConfig()
	proxyUrl, err := url.Parse(config.Proxy)
	if err != nil {
		log.Errorln("解析代理地址失败")
		return
	}
	sock5, _ := proxy.FromURL(proxyUrl, proxy.Direct)

	if sock5 == nil {
		log.Warningln("未配置代理，使用环境变量！")
		sock5 = proxy.FromEnvironmentUsing(proxy.Direct)
	}
	dc := sock5.(interface {
		DialContext(ctx context.Context, network, addr string) (net.Conn, error)
	})

	l, _ := zap.NewDevelopment(zap.IncreaseLevel(zapcore.InfoLevel), zap.AddStacktrace(zapcore.FatalLevel))
	client := telegram.NewClient(config.Auth.ApiId, config.Auth.ApiHash, telegram.Options{
		SessionStorage: &telegram.FileSessionStorage{Path: config.Auth.Type + ".session"},
		DC:             5,
		DialTimeout:    time.Minute * 5,
		Logger:         l,
		Resolver: dcs.Plain(dcs.PlainOptions{
			Dial: dc.DialContext,
		}),
		Middlewares: []telegram.Middleware{
			hook.UpdateHook(manager.Handle),
		},
		UpdateHandler: manager,
	})

	err = client.Run(ctx, func(ctx context.Context) error {
		if config.Auth.Type == "bot" {
			_, err := client.Auth().Bot(ctx, config.Auth.BotToken)
			if err != nil {
				log.Errorln("鉴权失败" + err.Error())
				return err
			}
		} else {
			if config.Auth.LoginType == "qrcode" {
				status, err := client.Auth().Status(ctx)
				if err != nil {
					log.Errorln("获取status失败")
					log.Errorln(err.Error())
					return err
				}
				if !status.Authorized {
					loginIn := make(chan struct{}, 1)
					go func() {
						log.Infoln("扫码后请按下回车！！")
						_, _ = fmt.Scanln()
						loginIn <- struct{}{}
					}()
					_, _ = qrlogin.NewQR(client.API(), config.Auth.ApiId, config.Auth.ApiHash, qrlogin.Options{}).Auth(ctx, loginIn, func(ctx context.Context, token qrlogin.Token) error {
						log.Infoln("二维码已生成到qr.png")
						_ = qrcode.WriteFile(token.URL(), qrcode.Medium, 255, "qr.png")
						qrcodeTerminal.New().Get(token.URL()).Print()
						return nil
					})

				}

			} else {
				err := client.Auth().IfNecessary(ctx, auth.NewFlow(&termAuth{}, auth.SendCodeOptions{}))
				if err != nil {
					log.Errorln("鉴权失败" + err.Error())
					return err
				}
			}
		}

		user, err := client.Self(ctx)
		if err != nil {
			log.Errorln(err.Error())
			return err
		}
		log.Infoln(fmt.Sprintf("%v已登陆", user.Username))
		// Notify update manager about authentication.

		if err := manager.Auth(context.WithValue(ctx, "client", client), client.API(), user.ID, user.Bot, true); err != nil {
			log.Errorln(err.Error())
			return err
		}

		// 处理连接事件
		onConnect(ctx, client)

		defer func() { _ = manager.Logout() }()
		<-ctx.Done()
		return ctx.Err()
	})
	if err != nil {
		log.Errorln(err.Error())
		return
	}
}

type loginToken struct {
}

func (l *loginToken) OnLoginToken(handler tg.LoginTokenHandler) {

}

// noSignUp can be embedded to prevent signing up.
type noSignUp struct{}

func (c noSignUp) SignUp(ctx context.Context) (auth.UserInfo, error) {
	return auth.UserInfo{}, errors.New("not implemented")
}

func (c noSignUp) AcceptTermsOfService(ctx context.Context, tos tg.HelpTermsOfService) error {
	return &auth.SignUpRequired{TermsOfService: tos}
}

// termAuth implements authentication via terminal.
type termAuth struct {
	noSignUp
}

func (a termAuth) Phone(_ context.Context) (string, error) {
	fmt.Print("Enter Phone: ")
	code, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(code), nil
}

func (a termAuth) Password(_ context.Context) (string, error) {
	fmt.Print("Enter 2FA password: ")
	bytePwd, err := terminal.ReadPassword(0)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(bytePwd)), nil
}

func (a termAuth) Code(_ context.Context, _ *tg.AuthSentCode) (string, error) {
	fmt.Print("Enter code: ")
	code, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(code), nil
}
