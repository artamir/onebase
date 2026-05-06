package interpreter

import "fmt"

// EmailSender is the minimal interface required by email DSL functions.
type EmailSender interface {
	Send(to, subject, textBody, htmlBody string) error
	Configured() bool
}

// ─── dslEmail (Новый ПисьмоEmail) ────────────────────────────────────────────

type dslEmail struct {
	sender  EmailSender
	to      string
	cc      string
	subject string
	text    string
	html    string
}

func (e *dslEmail) Get(field string) any {
	switch field {
	case "Кому", "To":
		return e.to
	case "Копия", "CC":
		return e.cc
	case "Тема", "Subject":
		return e.subject
	case "Текст", "Text", "Body":
		return e.text
	case "HTMLТело", "HTMLBody":
		return e.html
	}
	return nil
}

func (e *dslEmail) Set(field string, val any) {
	s := fmt.Sprintf("%v", val)
	switch field {
	case "Кому", "To":
		e.to = s
	case "Копия", "CC":
		e.cc = s
	case "Тема", "Subject":
		e.subject = s
	case "Текст", "Text", "Body":
		e.text = s
	case "HTMLТело", "HTMLBody":
		e.html = s
	}
}

func (e *dslEmail) CallMethod(name string, args []any) any {
	switch name {
	case "Отправить", "Send":
		if e.to == "" {
			panic(userError{Msg: "ПисьмоEmail.Отправить: поле Кому не задано"})
		}
		if e.subject == "" {
			panic(userError{Msg: "ПисьмоEmail.Отправить: поле Тема не задана"})
		}
		if err := e.sender.Send(e.to, e.subject, e.text, e.html); err != nil {
			panic(userError{Msg: "ОтправитьПисьмо: " + err.Error()})
		}
		return nil
	}
	panic(userError{Msg: "ПисьмоEmail: неизвестный метод " + name})
}

// ─── NewEmailFunctions ────────────────────────────────────────────────────────

// NewEmailFunctions returns DSL functions/factories to inject into extraVars.
// If sender is nil or not configured, functions panic with a user-friendly message.
func NewEmailFunctions(sender EmailSender) map[string]any {
	send := func(to, subject, textBody string) {
		if sender == nil || !sender.Configured() {
			panic(userError{Msg: "email не настроен — добавьте секцию email: в config/app.yaml"})
		}
		if err := sender.Send(to, subject, textBody, ""); err != nil {
			panic(userError{Msg: "ОтправитьПисьмо: " + err.Error()})
		}
	}

	shorthand := BuiltinFunc(func(args []any, file string, line int) (any, error) {
		to := strArg(args, 0)
		subject := strArg(args, 1)
		text := strArg(args, 2)
		send(to, subject, text)
		return nil, nil
	})

	emailFactory := func(args []any) any {
		if sender == nil || !sender.Configured() {
			panic(userError{Msg: "email не настроен — добавьте секцию email: в config/app.yaml"})
		}
		return &dslEmail{sender: sender}
	}

	return map[string]any{
		"ОтправитьПисьмо":     shorthand,
		"SendEmail":            shorthand,
		"__factory_ПисьмоEmail": emailFactory,
		"__factory_EmailMessage": emailFactory,
	}
}
