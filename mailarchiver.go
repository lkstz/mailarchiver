package main

import (
	"errors"
	"fmt"
	"github.com/ProtonMail/go-imap-id"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap-move"
	"github.com/emersion/go-imap/client"
	"golang.org/x/crypto/ssh/terminal"
	"gopkg.in/urfave/cli.v1"
	"log"
	"os"
	"strings"
	"syscall"
	"time"
)

const (
	version = "0.11.0"
)

var (
	app    *cli.App
	config *configuration

	mailboxes   []*mailbox
	supportMove bool

	cntMbox int
	cntMsg  int
)

type mailbox struct {
	Name      string
	Delimiter string
}

type configuration struct {
	Host string
	Port int
	User string
	Pass string

	Dry                bool
	SkipCurrent        bool
	Archive            string
	Mailboxes          []string
	RecursiveMailboxes []string
	IgnoreMailboxes    []string
}

type imapClient struct {
	*client.Client
	MoveClient *move.Client
	IdClient   *id.Client
}

func newImapClient(c *client.Client) *imapClient {
	return &imapClient{
		c,
		move.NewClient(c),
		id.NewClient(c),
	}
}

func (c *imapClient) Move(seqSet *imap.SeqSet, target string) (err error) {
	if supportMove {
		if err = c.MoveClient.UidMove(seqSet, target); err != nil {
			return
		}
	} else {
		if err = c.UidCopy(seqSet, target); err != nil {
			return
		}

		item := imap.FormatFlagsOp(imap.AddFlags, true)
		flags := []interface{}{imap.DeletedFlag}
		if err = c.UidStore(seqSet, item, flags, nil); err != nil {
			return
		}

		if err = c.Expunge(nil); err != nil {
			return
		}
	}

	return nil
}

func main() {
	app = cli.NewApp()
	app.Name = "Mailarchiver"
	app.Usage = "Automatically sort your emails into a year-month folder structure"
	app.Version = version
	app.Action = action

	config = &configuration{}

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "host, H",
			Usage:       "IMAP host",
			Destination: &config.Host,
		},
		cli.IntFlag{
			Name:        "port, p",
			Usage:       "IMAP port",
			Destination: &config.Port,
		},
		cli.StringFlag{
			Name:        "user, u",
			Usage:       "IMAP user",
			Destination: &config.User,
		},
		cli.StringFlag{
			Name:        "password, pw",
			Usage:       "IMAP password (will be promted, if omitted)",
			Destination: &config.Pass,
		},
		cli.StringFlag{
			Name:        "archive, a",
			Usage:       "Main archive folder",
			Destination: &config.Archive,
		},
		cli.StringSliceFlag{
			Name:  "mbox",
			Usage: "Mailboxes to process",
		},
		cli.StringSliceFlag{
			Name:  "rmbox",
			Usage: "Mailboxes to process recursive",
		},
		cli.StringSliceFlag{
			Name:  "imbox",
			Usage: "Mailboxes to ignore (overrides mbox and rmbox)",
		},
		cli.BoolFlag{
			Name:        "skip-current",
			Usage:       "Skip mails from current month",
			Destination: &config.SkipCurrent,
		},
		cli.BoolFlag{
			Name:        "dry",
			Usage:       "Performs a dry run, nothing will be changed on the IMAP server",
			Destination: &config.Dry,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func action(ctx *cli.Context) (err error) {
	// Config
	config.Mailboxes = ctx.StringSlice("mbox")
	config.RecursiveMailboxes = ctx.StringSlice("rmbox")
	config.IgnoreMailboxes = ctx.StringSlice("imbox")

	if len(config.Host) == 0 || config.Port < 1 || len(config.User) == 0 {
		log.Fatal(errors.New("no host, port or user supplied"))
	}

	if len(config.Mailboxes) == 0 && len(config.RecursiveMailboxes) == 0 {
		log.Fatal(errors.New("please supply at least one mailbox to process"))
	}

	// If no password was set via flag, promt the user for a password
	if len(config.Pass) == 0 {
		if err = promptPassword(); err != nil {
			return err
		}
	}

	if config.Dry {
		fmt.Println("DRY RUN")
	}

	c, err := getClient()
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	mailboxes, err = getMailboxes(c)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

mboxLoop:
	for _, m := range mailboxes {
		for _, iM := range config.IgnoreMailboxes {
			if strings.HasPrefix(m.Name, iM) {
				fmt.Printf("Ignore mailbox '%s'", m.Name)

				continue mboxLoop
			}
		}

		for _, cM := range config.Mailboxes {
			if m.Name == cM {
				processMailbox(c, m)
			}
		}

		for _, rM := range config.RecursiveMailboxes {
			if strings.HasPrefix(m.Name, rM) {
				if err = processMailbox(c, m); err != nil {
					fmt.Println(err.Error())
					return
				}
			}
		}
	}

	c.Logout()

	fmt.Printf("Done! Processed %d mailbox(es) and moved %d message(s)", cntMbox, cntMsg)
	if config.Dry {
		fmt.Println(" (DRY RUN)")
	} else {
		fmt.Println()
	}

	return nil
}

func getClient() (c *imapClient, err error) {
	// Connect to server
	fmt.Printf("Connecting to server %s:%d...", config.Host, config.Port)
	tmpC, err := client.DialTLS(fmt.Sprintf("%s:%d", config.Host, config.Port), nil)
	if err != nil {
		return
	}
	fmt.Println("  SUCCESS")

	// Login
	fmt.Printf("Login as user %s...", config.User)
	if err = tmpC.Login(config.User, config.Pass); err != nil {
		return
	}
	fmt.Println("  SUCCESS")

	c = newImapClient(tmpC)

	// Check once, if server supports move
	supportMove, err = c.MoveClient.SupportMove()
	if err != nil {
		return
	}

	// Send RFC2971 IMAP ID command
	if supportID, err := c.IdClient.SupportID(); err != nil && supportID {
		myId := id.ID{
			id.FieldName:       "mailarchiver",
			id.FieldSupportURL: "https://github.com/kolletzki/mailarchiver/issues",
			id.FieldVersion:    app.Version,
		}
		_, err = c.IdClient.ID(myId)
	}

	return
}

func getMailboxes(c *imapClient) (mailboxes []*mailbox, err error) {
	fmt.Print("Get all mailboxes...")

	mboxes := make(chan *imap.MailboxInfo, 10)
	done := make(chan error, 1)
	go func() {
		done <- c.List("", "*", mboxes)
	}()

	for m := range mboxes {
		mailboxes = append(mailboxes, &mailbox{
			Name:      m.Name,
			Delimiter: m.Delimiter,
		})
	}

	if err = <-done; err != nil {
		return
	}

	fmt.Println("  SUCCESS")
	return
}

func processMailbox(c *imapClient, mbox *mailbox) (err error) {
	fmt.Printf("Processing mailbox %s\n", mbox.Name)

	_, err = c.Select(mbox.Name, false)
	if err != nil {
		return
	}

	seqSet, err := imap.ParseSeqSet("1:*")
	if err != nil {
		return
	}

	msgChan := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	go func() {
		// TODO: Use "BODY.PEEK[HEADER.FIELDS (DATE)]" instead of fetching full envelope
		done <- c.UidFetch(seqSet, []imap.FetchItem{imap.FetchEnvelope}, msgChan)
	}()

	var messages []*imap.Message
	for msg := range msgChan {
		messages = append(messages, msg)
	}

	if err = <-done; err != nil {
		return
	}

	currentDate := time.Now().Local()

	for _, msg := range messages {
		var mboxTargetName string

		// Some messages may have no or invalid Date headers => sort these in the main archive folder
		if msg.Envelope.Date.Equal(time.Time{}) {
			mboxTargetName = config.Archive
		} else {
			msgDate := msg.Envelope.Date.Local()
			// Check if the message should be skipped because it is from the current month in the current year
			if config.SkipCurrent && currentDate.Year() == msgDate.Year() && currentDate.Month() == msgDate.Month() {
				continue
			}

			mboxTargetName = config.Archive + mbox.Delimiter + msg.Envelope.Date.Local().Format("2006"+mbox.Delimiter+"01")
		}

		mboxTarget := &mailbox{
			mboxTargetName,
			mbox.Delimiter,
		}

		if mbox.Name != mboxTarget.Name {
			fmt.Printf("Moving message %d to mailbox '%s'\n", msg.Uid, mboxTarget.Name)

			if err = mboxTarget.ensureAvailable(c); err != nil {
				return
			}

			// TODO: Enhance moving logic
			// Currently, each message is moved separately, which isn't ideal
			// Moving includes creating a new seqSet and calling MOVE, this is somehow expensive when done per message
			// It gets worse, if MOVE isn't available, then we're calling COPY, STORE and EXPUNGE for each message
			// In addition, this makes no sense, since we're already using a SeqSet, which is made for this situation
			// Optimally, we'd move (or copy, store and expunge) once per processed mailbox, to optimize performance

			if err = moveMessage(c, msg.Uid, mboxTarget); err != nil {
				return
			}
			cntMsg++
		}
	}

	cntMbox++
	return
}

func moveMessage(c *imapClient, uid uint32, mbox *mailbox) (err error) {
	seqSet, err := imap.ParseSeqSet(fmt.Sprint(uid))
	if err != nil {
		return
	}

	if !config.Dry {
		c.Move(seqSet, mbox.Name)
	}

	return
}

func (mbox *mailbox) ensureAvailable(c *imapClient) (err error) {
	parts := strings.Split(mbox.Name, mbox.Delimiter)

	for pI := range parts {
		exists := false
		target := ""

		for i := 0; i <= pI; i++ {
			if len(target) > 0 {
				target += mbox.Delimiter
			}
			target += parts[i]
		}

		for _, m := range mailboxes {
			if m.Name == target {
				exists = true
				break
			}
		}

		if !exists {
			fmt.Printf("Create mailbox %s", target)

			if !config.Dry {
				err = c.Create(target)
				if err != nil {
					return
				}
			}
			mailboxes = append(mailboxes, &mailbox{
				Name:      target,
				Delimiter: mbox.Delimiter,
			})
			fmt.Println("  SUCCESS")
		}
	}

	return
}

func promptPassword() (err error) {
	fmt.Print("Please enter password: ")

	bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return fmt.Errorf("error while reading password from stdin: %s", err.Error())
	}

	config.Pass = string(bytePassword)
	fmt.Println("")

	return nil
}
