package app

import (
	"strings"
	"testing"
)

func TestSetLocaleEnglish(t *testing.T) {
	SetLocale("en-US")
	defer SetLocale("en")

	if GetLocale() != "en" {
		t.Errorf("expected 'en', got %q", GetLocale())
	}
}

func TestSetLocaleKorean(t *testing.T) {
	SetLocale("ko-KR")
	defer SetLocale("en")

	if GetLocale() != "ko" {
		t.Errorf("expected 'ko', got %q", GetLocale())
	}
}

func TestSetLocaleKoreanShort(t *testing.T) {
	SetLocale("ko")
	defer SetLocale("en")

	if GetLocale() != "ko" {
		t.Errorf("expected 'ko', got %q", GetLocale())
	}
}

func TestSetLocaleUnknownFallsBackToEnglish(t *testing.T) {
	SetLocale("fr-FR")
	defer SetLocale("en")

	if GetLocale() != "en" {
		t.Errorf("expected 'en' for unknown locale, got %q", GetLocale())
	}
}

func TestSetLocaleEmpty(t *testing.T) {
	SetLocale("")
	defer SetLocale("en")

	if GetLocale() != "en" {
		t.Errorf("expected 'en' for empty locale, got %q", GetLocale())
	}
}

func TestTReturnsEnglishByDefault(t *testing.T) {
	SetLocale("en")
	defer SetLocale("en")

	msg := T(MsgInstalling)
	if !strings.Contains(msg, "%s") {
		t.Errorf("expected format string with %%s, got %q", msg)
	}
}

func TestTReturnsKorean(t *testing.T) {
	SetLocale("ko")
	defer SetLocale("en")

	msg := T(MsgInstalling)
	if !strings.Contains(msg, "설치") {
		t.Errorf("expected Korean message containing '설치', got %q", msg)
	}
}

func TestTRetryBodyKorean(t *testing.T) {
	SetLocale("ko")
	defer SetLocale("en")

	msg := T(MsgRetryBody)
	if !strings.Contains(msg, "종료") {
		t.Errorf("expected Korean retry body containing '종료', got %q", msg)
	}
}

func TestTRetryBodyEnglish(t *testing.T) {
	SetLocale("en")
	defer SetLocale("en")

	msg := T(MsgRetryBody)
	if !strings.Contains(msg, "Retry") {
		t.Errorf("expected English retry body containing 'Retry', got %q", msg)
	}
}

func TestAllMessageKeysHaveEnglish(t *testing.T) {
	SetLocale("en")
	defer SetLocale("en")

	keys := []MessageKey{
		MsgInstalling, MsgAlreadyInstalled, MsgInstalledContentChanged,
		MsgExtracting, MsgLaunching, MsgNoPackageFound, MsgRemovingOldVersion,
		MsgOldVersionInUse, MsgFailedToRemove, MsgRetryTitle,
		MsgRetryBody, MsgErrorTitle,
	}
	for _, key := range keys {
		msg := T(key)
		if msg == "" {
			t.Errorf("missing English message for key %d", key)
		}
	}
}

func TestAllMessageKeysHaveKorean(t *testing.T) {
	SetLocale("ko")
	defer SetLocale("en")

	keys := []MessageKey{
		MsgInstalling, MsgAlreadyInstalled, MsgInstalledContentChanged,
		MsgExtracting, MsgLaunching, MsgNoPackageFound, MsgRemovingOldVersion,
		MsgOldVersionInUse, MsgFailedToRemove, MsgRetryTitle,
		MsgRetryBody, MsgErrorTitle,
	}
	for _, key := range keys {
		msg := T(key)
		if msg == "" {
			t.Errorf("missing Korean message for key %d", key)
		}
	}
}
