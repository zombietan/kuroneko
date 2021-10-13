package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/PuerkitoBio/goquery"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	errColor   func(string, ...interface{}) string = color.HiYellowString
	countColor func(string, ...interface{}) string = color.HiYellowString
)

type exitCode int

const (
	normal exitCode = iota
	abnormal
)

func (c exitCode) Exit() {
	os.Exit(int(c))
}

func newRootCmd(newOut, newErr io.Writer, args []string) *cobra.Command {

	cmd := &cobra.Command{
		Use:           "kuroneko [flags] 伝票番号",
		Short:         "ヤマト運輸のステータス取得",
		SilenceErrors: true,
		SilenceUsage:  true,

		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				count := strconv.Itoa(len(args))
				return fmt.Errorf("accepts at most 1 arg(s), received %s", errColor(count))
			}

			if len(args) == 0 {
				return coloredError("伝票番号を入力してください")
			}

			return nil
		},

		PreRunE: func(cmd *cobra.Command, args []string) error {
			flagCount := cmd.Flags().NFlag()
			if flagCount > 1 {
				count := strconv.Itoa(flagCount)
				return fmt.Errorf("accepte at most 1 flag(s), received %s", errColor(count))
			}

			serial, err := cmd.Flags().GetString("serial")
			if err != nil {
				return err
			}

			i, err := strconv.Atoi(serial)
			if err != nil {
				return coloredError("連番を整数で入力してください")
			}

			if i < 1 || i > 10 {
				return coloredError("連番で取得できるのは 1~10件 までです")
			}

			return nil
		},

		RunE: func(cmd *cobra.Command, args []string) error {
			trackingNumber := args[0]
			tracker := newTracker(cmd)
			return tracker.track(trackingNumber)
		},
	}

	cmd.Flags().StringP("serial", "s", "1", "連番取得(10件まで)")
	cmd.SetArgs(args)
	cmd.SetOut(newOut)
	cmd.SetErr(newErr)

	return cmd
}

func Execute(newOut, newErr io.Writer, args []string) exitCode {
	cmd := newRootCmd(newOut, newErr, args)
	if err := cmd.Execute(); err != nil {
		cmd.PrintErrf("Error: %+v\n", err)
		return abnormal
	}
	return normal
}

func init() {}

func makeSpace(count int) string {
	// 注:全角スペース
	s := "　"
	return strings.Repeat(s, count)
}

type tracker interface {
	track(s string) error
}

func newTracker(cmd *cobra.Command) tracker {
	flagCount := cmd.Flags().NFlag()
	switch flagCount {
	case 0:
		return &trackShipmentsOne{
			cmd: cmd,
		}
	default:
		// PreRunEでエラーチェック済み
		s, _ := cmd.Flags().GetString("serial")
		serial, _ := strconv.Atoi(s)
		return &trackShipmentsMultiple{
			cmd:    cmd,
			serial: serial,
		}
	}
}

type trackShipmentsOne struct {
	cmd *cobra.Command
}

func (t *trackShipmentsOne) track(s string) error {
	values := url.Values{}
	values.Add("number01", s)

	contactUrl := "http://toi.kuronekoyamato.co.jp/cgi-bin/tneko"
	http.DefaultClient.Timeout = 10 * time.Second
	resp, err := http.PostForm(contactUrl, values)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return err
	}

	w := t.cmd.OutOrStdout()
	trackNumber := doc.Find(".tracking-invoice-block-title").Text()
	trackNumber = strings.Replace(
		trackNumber, "1件目：", "伝票番号 ", -1,
	)
	stateTitle := doc.Find(".tracking-invoice-block-state-title").Text()
	stateSummary := doc.Find(".tracking-invoice-block-state-summary").Text()
	fmt.Fprintf(w, "%s\n%s\n%s\n", trackNumber, stateTitle, stateSummary)

	informations := doc.Find(".tracking-invoice-block-summary ul li").Map(
		func(_ int, s *goquery.Selection) string {
			item := s.Find(".item").Text()
			data := s.Find(".data").Text()
			return item + data
		},
	)
	if len(informations) != 0 {
		fmt.Fprintf(w, "\n")
		for _, info := range informations {
			fmt.Fprintf(w, "%s\n", info)
		}
		fmt.Fprintf(w, "\n")
	}

	doc.Find(".tracking-invoice-block-detail ol li").Each(func(_ int, args *goquery.Selection) {
		item := args.Find(".item").Text()
		itemLength := utf8.RuneCountInString(item)
		whitespace := 15 - itemLength
		space := makeSpace(whitespace)
		item = item + space
		date := args.Find(".date").Text()
		if date == "" {
			date = "              "
		}
		name := args.Find(".name").Text()
		nameLength := utf8.RuneCountInString(name)
		whitespace = 20 - nameLength
		space = makeSpace(whitespace)
		name = name + space
		fmt.Fprintf(w, "%s| %s | %s|\n", item, date, name)
	})

	underLine := strings.Repeat("-", 90)
	fmt.Fprintf(w, "%s\n", underLine)

	return nil
}

type trackShipmentsMultiple struct {
	cmd    *cobra.Command
	serial int
}

func (t *trackShipmentsMultiple) track(s string) error {
	trackingNumber := removeHyphen(s)
	if !isInt(trackingNumber) {
		return coloredError("不正な数値です")
	}

	if !is12or11Digits(trackingNumber) {
		return coloredError("12 or 11桁の伝票番号を入力してください")
	}

	if !isCorrectNumber(trackingNumber) {
		return coloredError("伝票番号に誤りがあります")
	}

	ctx, cancel := context.WithCancel(context.Background())
	ch := sevenCheckCalculate(ctx, trackingNumber[:len(trackingNumber)-1])
	values := url.Values{}

	var i int
	for i = 0; i < t.serial; i++ {
		querykey := fmt.Sprintf("number%02d", i+1)
		values.Add(querykey, <-ch)
	}
	cancel()

	contactUrl := "http://toi.kuronekoyamato.co.jp/cgi-bin/tneko"
	http.DefaultClient.Timeout = 30 * time.Second
	resp, err := http.PostForm(contactUrl, values)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return err
	}

	w := t.cmd.OutOrStdout()
	doc.Find(".parts-tracking-invoice-block").Each(func(i int, args *goquery.Selection) {
		count := strconv.Itoa(i+1) + "件目"
		trackNumber := args.Find(".tracking-invoice-block-title").Text()
		rep := regexp.MustCompile(`[0-9]*件目：`)
		trackNumber = rep.ReplaceAllString(trackNumber, "伝票番号 ")
		stateTitle := args.Find(".tracking-invoice-block-state-title").Text()
		stateSummary := args.Find(".tracking-invoice-block-state-summary").Text()
		fmt.Fprintf(w, "%s\n%s\n%s\n%s\n", countColor(count), trackNumber, stateTitle, stateSummary)

		informations := args.Find(".tracking-invoice-block-summary ul li").Map(
			func(_ int, s *goquery.Selection) string {
				item := s.Find(".item").Text()
				data := s.Find(".data").Text()
				return item + data
			},
		)
		if len(informations) != 0 {
			fmt.Fprintf(w, "\n")
			for _, info := range informations {
				fmt.Fprintf(w, "%s\n", info)
			}
			fmt.Fprintf(w, "\n")
		}

		args.Find(".tracking-invoice-block-detail ol li").Each(func(_ int, s *goquery.Selection) {
			item := s.Find(".item").Text()
			itemLength := utf8.RuneCountInString(item)
			whitespace := 15 - itemLength
			space := makeSpace(whitespace)
			item = item + space
			date := s.Find(".date").Text()
			if date == "" {
				date = "              "
			}
			name := s.Find(".name").Text()
			nameLength := utf8.RuneCountInString(name)
			whitespace = 20 - nameLength
			space = makeSpace(whitespace)
			name = name + space
			fmt.Fprintf(w, "%s| %s | %s|\n", item, date, name)
		})
		underLine := strings.Repeat("-", 90)
		fmt.Fprintf(w, "%s\n", underLine)
	})

	return nil
}

func coloredError(s string) error {
	return errors.New(errColor(s))
}

func removeHyphen(s string) string {
	if strings.Contains(s, "-") {
		removed := strings.Replace(s, "-", "", -1)
		return removed
	}
	return s
}

func sevenCheckCalculate(ctx context.Context, n string) <-chan string {
	ch := make(chan string)
	const coef = 7
	var format = "%012s"
	if len(n) == 10 {
		format = "%011s"
	}
	go func() {
		sign, _ := strconv.ParseInt(n, 10, 64)
	LOOP:
		for {
			select {
			case <-ctx.Done():
				break LOOP
			default:
				digit := sign % coef
				digitStr := strconv.FormatInt(digit, 10)
				trackingNumber := strconv.FormatInt(sign, 10) + digitStr
				zeroPaddingNumber := fmt.Sprintf(format, trackingNumber)
				ch <- zeroPaddingNumber
				sign++
			}
		}
		close(ch)
	}()
	return ch
}

func isCorrectNumber(s string) bool {
	const coef = 7
	lastDigits := s[len(s)-1:]
	otherDigits := s[:len(s)-1]
	sign, _ := strconv.ParseInt(otherDigits, 10, 64)
	digit := sign % coef
	return lastDigits == fmt.Sprint(digit)
}

func isInt(s string) bool {
	for _, c := range s {
		if !unicode.IsDigit(c) {
			return false
		}
	}
	return true
}

func is12or11Digits(s string) bool {
	if len(s) == 12 || len(s) == 11 {
		return true
	}
	return false
}
