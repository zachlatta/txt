package main

import (
	"errors"
	"flag"
	"io"
	"os"
	"strings"
	"sync"
	"text/template"

	"github.com/rakyll/pb"
	"github.com/subosito/twilio"
)

type Msg struct {
	Body      string
	Recipient string
}

type numbers struct {
	numbers []string
}

func (n *numbers) String() string {
	return strings.Join(n.numbers, ",")
}

func (n *numbers) Set(v string) error {
	n.numbers = strings.Split(v, ",")
	if len(n.numbers) == 0 {
		return errors.New("invalid input")
	}
	return nil
}

type result struct {
	err        error
	msg        *twilio.Message
	statusCode int
}

type Txter struct {
	Twilio *twilio.Client

	Senders    []string
	Recipients []string

	bar *pb.ProgressBar
	//rpt *report
	results chan *result
}

func newPb(size int) (bar *pb.ProgressBar) {
	bar = pb.New(size)
	bar.Start()
	return
}

func (t *Txter) Run() {
	t.bar = newPb(len(t.Recipients))
	t.run()
}

func (t *Txter) run() {
	jobs := make(chan *Msg, len(recipientNumbers.numbers))
	t.results = make(chan *result)

	var wg sync.WaitGroup
	wg.Add(len(t.Senders))

	// fire up workers
	for _, v := range t.Senders {
		go func() {
			t.worker(v, jobs)
			wg.Done()
		}()
	}

	// send jobs
	for _, v := range t.Recipients {
		m := &Msg{Recipient: v, Body: msg}
		jobs <- m
	}
	close(jobs)

	wg.Wait()
	if t.bar != nil {
		t.bar.Finish()
	}
}

func (t *Txter) worker(senderNumber string, jobs chan *Msg) {
	for msg := range jobs {
		msg, resp, err := twl.Messages.SendSMS(senderNumber, msg.Recipient, msg.Body)
		if err != nil {
			t.results <- &result{
				err:        err,
				msg:        nil,
				statusCode: resp.StatusCode,
			}
			continue
		}

		if t.bar != nil {
			t.bar.Increment()
		}

		t.results <- &result{
			err:        nil,
			msg:        msg,
			statusCode: resp.StatusCode,
		}
	}
}

var (
	msg, twilioSid, twilioAuthToken  string
	sendingNumbers, recipientNumbers numbers

	flags, requiredFlags []*flag.Flag

	twl *twilio.Client
)

func main() {
	flag.StringVar(&msg, "msg", "", "message to send")
	flag.StringVar(&twilioSid, "sid", "", "twilio sid")
	flag.StringVar(&twilioAuthToken, "token", "", "twilio auth token")
	flag.Var(&sendingNumbers, "sending", "twilio numbers to send from")
	flag.Var(&recipientNumbers, "recipients", "phone numbers to send to")

	requiredFlagNames := []string{"msg", "sid", "token", "sending", "recipients"}
	flag.VisitAll(func(f *flag.Flag) {
		flags = append(flags, f)
		for _, name := range requiredFlagNames {
			if name == f.Name {
				requiredFlags = append(requiredFlags, f)
			}
		}
	})

	flag.Usage = usage
	flag.Parse()

	var flagsMissing []*flag.Flag
	for _, f := range requiredFlags {
		if f.Value.String() == "" {
			flagsMissing = append(flagsMissing, f)
		}
	}
	missingCount := len(flagsMissing)
	if missingCount > 0 {
		if missingCount == len(requiredFlags) {
			usage()
		}
		missingFlags(flagsMissing)
	}

	txter := Txter{
		Twilio:     twilio.NewClient(twilioSid, twilioAuthToken, nil),
		Senders:    sendingNumbers.numbers,
		Recipients: recipientNumbers.numbers,
	}
	txter.Run()
}

const usageTemplate = `txt is a utility for sending lots of text messages to lots of phone numbers

Usage:

  txt [flags]

Flags:
{{range .}}
  -{{.Name | printf "%-11s"}} {{.Usage}}{{end}}

`

const missingFlagsTemplate = `Missing required flags:
{{range .}}
  -{{.Name | printf "%-11s"}} {{.Usage}}{{end}}

`

func tmpl(w io.Writer, text string, data interface{}) {
	t := template.New("top")
	template.Must(t.Parse(text))
	if err := t.Execute(w, data); err != nil {
		panic(err)
	}
}
func printUsage(w io.Writer) {
	tmpl(w, usageTemplate, flags)
}

func usage() {
	printUsage(os.Stderr)
	os.Exit(2)
}

func printMissingFlags(w io.Writer, missingFlags []*flag.Flag) {
	tmpl(w, missingFlagsTemplate, missingFlags)
}

func missingFlags(missingFlags []*flag.Flag) {
	printMissingFlags(os.Stderr, missingFlags)
	os.Exit(2)
}
