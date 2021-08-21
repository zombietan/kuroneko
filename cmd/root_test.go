package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestManyArgs(t *testing.T) {
	outStream := new(bytes.Buffer)
	cmd := newRootCmd()
	cmd.SetOut(outStream)

	cmd.SetArgs([]string{"00000000000", "000000000000"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("failed test1 ManyArgs")
	}
	errMessage := "accepts at most 1 arg(s), received 2"
	if err := cmd.Execute(); !strings.Contains(err.Error(), errMessage) {
		t.Fatal("failed test2 ManyArgs")
	}
}

func TestNoArgs(t *testing.T) {
	outStream := new(bytes.Buffer)
	cmd := newRootCmd()
	cmd.SetOut(outStream)

	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err == nil {
		t.Fatal("failed test1 NoArgs")
	}
	errMessage := "伝票番号を入力してください"
	if err := cmd.Execute(); !strings.Contains(err.Error(), errMessage) {
		t.Fatal("failed test2 NoArgs")
	}
}

func TestOverSerial(t *testing.T) {
	outStream := new(bytes.Buffer)
	cmd := newRootCmd()
	cmd.SetOut(outStream)

	cmd.SetArgs([]string{"-s", "100", "00000000000"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("failed test1 LimitOverSerial")
	}
	errMessage := "連番で取得できるのは 1~10件 までです"
	if err := cmd.Execute(); !strings.Contains(err.Error(), errMessage) {
		t.Fatal("failed test2 LimitOverSerial")
	}
}

func TestFewSerial(t *testing.T) {
	outStream := new(bytes.Buffer)
	cmd := newRootCmd()
	cmd.SetOut(outStream)

	cmd.SetArgs([]string{"-s", "0", "00000000000"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("failed test1 FewSerial")
	}
	errMessage := "連番で取得できるのは 1~10件 までです"
	if err := cmd.Execute(); !strings.Contains(err.Error(), errMessage) {
		t.Fatal("failed test2 FewSerial")
	}
}
