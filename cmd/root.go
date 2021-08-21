package cmd

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/PuerkitoBio/goquery"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

var errColor func(string, ...interface{}) string = color.HiYellowString
var countColor func(string, ...interface{}) string = color.HiYellowString

func newRootCmd() *cobra.Command {
	var f int

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
				return fmt.Errorf("%s", errColor("伝票番号を入力してください"))
			}

			return nil
		},

		PreRunE: func(cmd *cobra.Command, args []string) error {
			flagCount := cmd.Flags().NFlag()
			if flagCount > 1 {
				count := strconv.Itoa(flagCount)
				return fmt.Errorf("accepte at most 1 flag(s), received %s", errColor(count))
			}

			if f < 1 || f > 10 {
				return fmt.Errorf("%s", errColor("連番で取得できるのは 1~10件 までです"))
			}

			return nil
		},

		RunE: func(cmd *cobra.Command, args []string) error {
			trackingNumber := args[0]
			flagCount := cmd.Flags().NFlag()
			if flagCount == 0 {
				return TrackShipmentsOne(trackingNumber, cmd.OutOrStdout())
			}

			return TrackShipmentsMultiple(trackingNumber, f, cmd.OutOrStdout())
		},
	}

	cmd.Flags().IntVarP(&f, "serial", "s", 1, "連番取得(10件まで)")
	cmd.SetOut(color.Output)
	cmd.SetErr(color.Error)

	return cmd
}

func Execute() {
	cmd := newRootCmd()
	if err := cmd.Execute(); err != nil {
		w := cmd.ErrOrStderr()
		fmt.Fprintf(w, "Error: %+v\n", err)
		os.Exit(1)
	}
}

func init() {}

func makeSpace(count int) string {
	// 注:全角スペース
	s := "　"
	return strings.Repeat(s, count)
}

var TrackShipmentsOne = func(s string, w io.Writer) error {
	values := url.Values{}
	values.Add("number00", "1")
	values.Add("number01", s)

	contactUrl := "http://toi.kuronekoyamato.co.jp/cgi-bin/tneko"
	resp, err := http.PostForm(contactUrl, values)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	utfBody := transform.NewReader(bufio.NewReader(resp.Body), japanese.ShiftJIS.NewDecoder())

	doc, err := goquery.NewDocumentFromReader(utfBody)
	if err != nil {
		return err
	}

	doc.Find(".saisin td").Each(func(_ int, args *goquery.Selection) {
		if args.HasClass("bold") || args.HasClass("font14") {
			text := args.Text()
			fmt.Fprintf(w, " %s\n", text)
		}
	})

	fmt.Fprintf(w, "\n")

	doc.Find(".meisai tr").Each(func(i int, args *goquery.Selection) {
		if i != 0 {
			information := args.Find("td").Map(func(_ int, s *goquery.Selection) string {
				text := s.Text()
				return text
			})
			detailInfo := information[1:6]
			statusLength := utf8.RuneCountInString(detailInfo[0])
			whitespace := 15 - statusLength
			space := makeSpace(whitespace)
			status := detailInfo[0] + space
			branchLength := utf8.RuneCountInString(detailInfo[3])
			whitespace = 20 - branchLength
			space = makeSpace(whitespace)
			branch := detailInfo[3] + space
			date, times, code := detailInfo[1], detailInfo[2], detailInfo[4]
			fmt.Fprintf(w, " %s| %s | %s | %s| %s |\n", status, date, times, branch, code)
		}
	})

	underLine := strings.Repeat("-", 99)
	fmt.Fprintf(w, "%s\n", underLine)

	return nil

}

var TrackShipmentsMultiple = func(s string, c int, w io.Writer) error {
	trackingNumber := removeHyphen(s)
	if !isInt(trackingNumber) {
		return fmt.Errorf("%s", errColor("不正な数値です"))
	}

	if !is12or11Digits(trackingNumber) {
		return fmt.Errorf("%s", errColor("12 or 11桁の伝票番号を入力してください"))
	}

	if !isCorrectNumber(trackingNumber) {
		return fmt.Errorf("%s", errColor("伝票番号に誤りがあります"))
	}

	ch := sevenCheckCalculate(trackingNumber[:len(trackingNumber)-1])
	values := url.Values{}
	values.Add("number00", "1")

	var i int
	for i = 0; i < c; i++ {
		querykey := fmt.Sprintf("number%02d", i+1)
		values.Add(querykey, <-ch)
	}

	contactUrl := "http://toi.kuronekoyamato.co.jp/cgi-bin/tneko"
	resp, err := http.PostForm(contactUrl, values)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	utfBody := transform.NewReader(bufio.NewReader(resp.Body), japanese.ShiftJIS.NewDecoder())

	doc, err := goquery.NewDocumentFromReader(utfBody)
	if err != nil {
		return err
	}

	doc.Find("center").Each(func(_ int, s *goquery.Selection) {
		hasDetail := false
		s.Find(".saisin td").Each(func(_ int, args *goquery.Selection) {
			if args.HasClass("number") {
				hasDetail = true
				subject := args.Text()
				fmt.Fprintf(w, " %s\n", countColor(subject))
			}

			if args.HasClass("bold") || args.HasClass("font14") {
				text := args.Text()
				fmt.Fprintf(w, " %s\n", text)
			}
		})

		if hasDetail {
			fmt.Fprintf(w, "\n")
		}

		s.Find(".meisai tr").Each(func(i int, args *goquery.Selection) {
			if i != 0 {
				information := args.Find("td").Map(func(_ int, s *goquery.Selection) string {
					text := s.Text()
					return text
				})
				detailInfo := information[1:6]
				statusLength := utf8.RuneCountInString(detailInfo[0])
				whitespace := 15 - statusLength
				space := makeSpace(whitespace)
				status := detailInfo[0] + space
				branchLength := utf8.RuneCountInString(detailInfo[3])
				whitespace = 20 - branchLength
				space = makeSpace(whitespace)
				branch := detailInfo[3] + space
				date, times, code := detailInfo[1], detailInfo[2], detailInfo[4]
				fmt.Fprintf(w, " %s| %s | %s | %s| %s |\n", status, date, times, branch, code)
			}
		})

		if hasDetail {
			underLine := strings.Repeat("-", 99)
			fmt.Fprintf(w, "%s\n", underLine)
		}
	})

	return nil
}

func removeHyphen(s string) string {
	if strings.Contains(s, "-") {
		removed := strings.Replace(s, "-", "", -1)
		return removed
	}
	return s
}

func sevenCheckCalculate(n string) <-chan string {
	ch := make(chan string)
	const coef = 7
	var format = "%012s"
	if len(n) == 10 {
		format = "%011s"
	}
	go func() {
		sign, _ := strconv.ParseInt(n, 10, 64)
		for {
			digit := sign % coef
			digitStr := strconv.FormatInt(digit, 10)
			trackingNumber := strconv.FormatInt(sign, 10) + digitStr
			zeroPaddingNumber := fmt.Sprintf(format, trackingNumber)
			ch <- zeroPaddingNumber
			sign++
		}
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
