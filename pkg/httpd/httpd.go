package httpd

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/happycakefriends/certainly/pkg/certainly"
	"github.com/happycakefriends/certainly/pkg/notification"
	"github.com/happycakefriends/certainly/pkg/util"
	"go.uber.org/zap"
)

type HTTPD struct {
	Config       *certainly.CertainlyCFG
	Logger       *zap.SugaredLogger
	errChan      chan error
	Notification *notification.Notifications
}

func InitAndStart(config *certainly.CertainlyCFG, tlsconfig *tls.Config, logger *zap.SugaredLogger, notification *notification.Notifications, errChan chan error) *HTTPD {
	h := &HTTPD{
		Config:       config,
		Logger:       logger,
		errChan:      errChan,
		Notification: notification,
	}
	go h.startHTTP(config, logger, notification, errChan)
	go h.startHTTPS(tlsconfig, config, logger, notification, errChan)
	return h
}

func (h *HTTPD) ShouldInjectTemplate(req *http.Request) (bool, string) {
	for _, filter := range h.Config.HTTPD.InjectionFilters {
		match, err := regexp.MatchString(filter, req.RequestURI)
		if err == nil && match {
			return false, ""
		}
	}
	for rule, templatef := range h.Config.HTTPDInjections {
		match, err := regexp.MatchString(rule, req.RequestURI)
		if err == nil && match {
			return true, templatef
		}
	}
	return false, ""
}

func (h *HTTPD) MakeProxyRequest(req *http.Request, proto string) (*http.Response, error) {
	url := req.URL
	url.Scheme = proto
	url.Host = util.ReplaceApex(req.Host, h.Config.Rewrites)
	newReq, err := http.NewRequest(req.Method, req.URL.String(), req.Body)
	if err != nil {
		return nil, err
	}
	for header, values := range req.Header {
		for _, value := range values {
			if strings.ToLower(header) == "host" {
				value = util.ReplaceApex(value, h.Config.Rewrites)
			}
			newReq.Header.Add(header, value)
		}
	}
	newReq.Body = req.Body
	client := &http.Client{}
	return client.Do(newReq)
}

func (h *HTTPD) copyHeaders(from http.Header, to http.Header) {
	for header, values := range from {
		if strings.ToLower(header) == "content-length" {
			continue
		}
		for _, value := range values {
			to.Set(header, value)
		}
	}
}

func (h *HTTPD) injectTemplate(templateFile, data, hash string) string {
	templateFilePath := filepath.Join(h.Config.HTTPD.InjectionTemplateFilepath, templateFile)
	templateData, err := os.ReadFile(templateFilePath)
	if err != nil {
		h.Logger.Errorw("Could not read template file", "error", err)
		return data
	}
	data = strings.ReplaceAll(string(templateData), "CERTAINLY_UPSTREAM", data)
	return strings.ReplaceAll(data, "CERTAINLY_HASH", hash)
}

func (h *HTTPD) startHTTPS(tlsconfig *tls.Config, config *certainly.CertainlyCFG, sugar *zap.SugaredLogger, notification *notification.Notifications, errChan chan error) {
	stderrorlog, err := zap.NewStdLogAt(sugar.Desugar(), zap.ErrorLevel)
	if err != nil {
		errChan <- err
		return
	}

	tlsconfig.NextProtos = append([]string{"http/1.1", "h2", "http/1.0"}, tlsconfig.NextProtos...)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uuid := uuid.New().String()
		res, err := httputil.DumpRequest(r, true)
		if err != nil {
			sugar.Error(err)
		}
		notification.Notify("http", fmt.Sprintf(`
Inbound HTTPS request %s from: %s

%s`, uuid, r.RemoteAddr, string(res)))
		sugar.Infow(
			"Inbound HTTPS request",
			"request", string(res),
			"remoteAddr", r.RemoteAddr,
			"uuid", uuid)
		shouldInject, templateFile := h.ShouldInjectTemplate(r)
		if shouldInject {
			proxyResp, err := h.MakeProxyRequest(r, "https")
			if err != nil {
				sugar.Errorw("Could not make proxy request", "error", err)
			} else {
				defer proxyResp.Body.Close()
				bodyBytes, err := io.ReadAll(proxyResp.Body)
				if err != nil {
					sugar.Errorw("Could not read proxy response body", "error", err)
					return
				}
				h.copyHeaders(proxyResp.Header, w.Header())
				w.WriteHeader(proxyResp.StatusCode)
				w.Write([]byte(h.injectTemplate(templateFile, string(bodyBytes), uuid)))
				return
			}
		} else {
			for source, target := range config.Rewrites {
				if strings.Contains(r.Host, source) {
					target := "https://" + strings.Replace(r.Host, source, target, 1) + r.URL.Path
					if len(r.URL.RawQuery) > 0 {
						target += "?" + r.URL.RawQuery
					}
					http.Redirect(w, r, target, http.StatusTemporaryRedirect)
					return
				}
			}
		}
		fmt.Fprintf(w, "Excellent choice, sir!")
	})

	srv := &http.Server{
		Addr:      ":443",
		Handler:   handler,
		TLSConfig: tlsconfig,
		ErrorLog:  stderrorlog,
	}
	errChan <- srv.ListenAndServeTLS("", "")
}

func (h *HTTPD) startHTTP(config *certainly.CertainlyCFG, sugar *zap.SugaredLogger, notification *notification.Notifications, errChan chan error) {
	stderrorlog, err := zap.NewStdLogAt(sugar.Desugar(), zap.ErrorLevel)
	if err != nil {
		errChan <- err
		return
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uuid := uuid.New().String()
		res, err := httputil.DumpRequest(r, true)
		if err != nil {
			sugar.Error(err)
		}
		notification.Notify("http", fmt.Sprintf(`
Inbound plaintext HTTP request from: %s

%s`, r.RemoteAddr, string(res)))

		sugar.Infow(
			"Inbound HTTP request",
			"request", string(res),
			"remoteAddr", r.RemoteAddr,
			"uuid", uuid)

		shouldInject, templateFile := h.ShouldInjectTemplate(r)
		if shouldInject {
			proxyResp, err := h.MakeProxyRequest(r, "http")
			if err != nil {
				sugar.Errorw("Could not make proxy request", "error", err)
			} else {
				defer proxyResp.Body.Close()
				bodyBytes, err := io.ReadAll(proxyResp.Body)
				if err != nil {
					sugar.Errorw("Could not read proxy response body", "error", err)
					return
				}
				h.copyHeaders(proxyResp.Header, w.Header())
				w.WriteHeader(proxyResp.StatusCode)
				w.Write([]byte(h.injectTemplate(templateFile, string(bodyBytes), uuid)))
				return
			}
		} else {
			for source, target := range config.Rewrites {
				if strings.Contains(r.Host, source) {
					target := "http://" + strings.Replace(r.Host, source, target, 1) + r.URL.Path
					if len(r.URL.RawQuery) > 0 {
						target += "?" + r.URL.RawQuery
					}
					http.Redirect(w, r, target, http.StatusTemporaryRedirect)
					return
				}
			}
		}
		fmt.Fprintf(w, "Excellent choice, sir!")
	})

	srv := &http.Server{
		Addr:     ":80",
		Handler:  handler,
		ErrorLog: stderrorlog,
	}
	errChan <- srv.ListenAndServe()
}
