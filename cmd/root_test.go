package cmd

import (
	"bytes"
	"testing"

	"github.com/fatih/color"
)

func TestManyArgs(t *testing.T) {
	color.NoColor = true
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	ext := Execute(outBuf, errBuf, []string{"00000000000", "000000000000"})
	if ext != abnormal {
		t.Fatal("failed test1 ManyArgs")
	}
	errMessage := "Error: accepts at most 1 arg(s), received 2\n"
	if errMessage != errBuf.String() {
		t.Fatal("failed test2 ManyArgs")
	}
}

func TestNoArgs(t *testing.T) {
	color.NoColor = true
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	ext := Execute(outBuf, errBuf, []string{})
	if ext != abnormal {
		t.Fatal("failed test1 NoArgs")
	}
	errMessage := "Error: 伝票番号を入力してください\n"
	if errMessage != errBuf.String() {
		t.Fatal("failed test2 NoArgs")
	}
}

func TestTooManySerial(t *testing.T) {
	color.NoColor = true
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	ext := Execute(outBuf, errBuf, []string{"--serial", "11", "00000000000"})
	if ext != abnormal {
		t.Fatal("failed test1 OvreSerial")
	}
	errMessage := "Error: 連番で取得できるのは 1~10件 までです\n"
	if errMessage != errBuf.String() {
		t.Fatal("failed test2 OverSerial")
	}
}

func TestNotEnoughSerial(t *testing.T) {
	color.NoColor = true
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	ext := Execute(outBuf, errBuf, []string{"--serial", "0", "00000000000"})
	if ext != abnormal {
		t.Fatal("failed test1 FewSerial")
	}
	errMessage := "Error: 連番で取得できるのは 1~10件 までです\n"
	if errMessage != errBuf.String() {
		t.Fatal("failed test2 FewSerial")
	}
}

func TestRemoveHyphen(t *testing.T) {
	var tests = []struct {
		input    string
		expected string
	}{
		{"foo-", "foo"},
		{"-bar", "bar"},
		{"baz", "baz"},
		{"000-0000-0000", "00000000000"},
	}

	for _, test := range tests {
		output := removeHyphen(test.input)
		if output != test.expected {
			t.Errorf("Test Faild: input %v, got %v, want %v", test.input, output, test.expected)
		}
	}

}

func TestIsInt(t *testing.T) {
	var tests = []struct {
		input    string
		expected bool
	}{
		{"123456789012", true},
		{"00000000000a", false},
		{"000-000-0000", false},
		{"00.000000000", false},
		{"000000000000", true},
		{"000000000001", true},
		{"00000000000１", true},
	}

	for _, test := range tests {
		output := isInt(test.input)
		if output != test.expected {
			t.Errorf("Test Faild: input %v, got %v, want %v", test.input, output, test.expected)
		}
	}
}

func TestIs12or11Digits(t *testing.T) {
	var tests = []struct {
		input    string
		expected bool
	}{
		{"123456789012", true},
		{"12345678901", true},
		{"００００００００００００", false},
		{"０００００００００００", false},
	}

	for _, test := range tests {
		output := is12or11Digits(test.input)
		if output != test.expected {
			t.Errorf("Test Faild: input %v, got %v, want %v", test.input, output, test.expected)
		}
	}
}
