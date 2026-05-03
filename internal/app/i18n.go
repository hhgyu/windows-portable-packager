package app

import (
	"strings"
)

type locale int

const (
	localeEN locale = iota
	localeKO
)

type MessageKey int

const (
	MsgInstalling MessageKey = iota
	MsgAlreadyInstalled
	MsgExtracting
	MsgLaunching
	MsgNoPackageFound
	MsgRemovingOldVersion
	MsgOldVersionInUse
	MsgFailedToRemove
	MsgRetryTitle
	MsgRetryBody
	MsgErrorTitle
	MsgFilesLocked
)

var messages = map[locale]map[MessageKey]string{
	localeEN: {
		MsgInstalling:         "Installing %s %s...",
		MsgAlreadyInstalled:   "Already installed and verified, launching...",
		MsgExtracting:         "Extracting to %s...",
		MsgLaunching:          "Launching %s",
		MsgNoPackageFound:     "No package found, looking for installed version...",
		MsgRemovingOldVersion: "Removing old version: %s",
		MsgOldVersionInUse:    "Previous version %s is still in use. Waiting 3 seconds... (attempt %d/%d)",
		MsgFailedToRemove:     "Warning: failed to remove old version %s: %v",
		MsgRetryTitle:         "%s - Update",
		MsgRetryBody:          "Previous version %s is still running.\n\nPlease close it and click Retry to continue.",
		MsgErrorTitle:         "%s - Error",
		MsgFilesLocked:        "App is running. Please close the following and retry:\n%s",
	},
	localeKO: {
		MsgInstalling:         "%s %s 설치 중...",
		MsgAlreadyInstalled:   "이미 설치되어 있습니다. 실행합니다...",
		MsgExtracting:         "%s 에 압축 해제 중...",
		MsgLaunching:          "%s 실행 중",
		MsgNoPackageFound:     "패키지를 찾을 수 없습니다. 설치된 버전을 찾는 중...",
		MsgRemovingOldVersion: "이전 버전 제거 중: %s",
		MsgOldVersionInUse:    "이전 버전 %s 이(가) 실행 중입니다. 3초 후 재시도합니다... (%d/%d)",
		MsgFailedToRemove:     "경고: 이전 버전 %s 제거 실패: %v",
		MsgRetryTitle:         "%s - 업데이트",
		MsgRetryBody:          "이전 버전 %s 이(가) 실행 중입니다.\n\n종료 후 재시도를 클릭하세요.",
		MsgErrorTitle:         "%s - 오류",
		MsgFilesLocked:        "앱이 실행 중입니다. 다음을 종료한 뒤 재시도하세요:\n%s",
	},
}

var currentLocale locale = localeEN

func SetLocale(lang string) {
	lang = strings.ToLower(strings.TrimSpace(lang))
	if strings.HasPrefix(lang, "ko") {
		currentLocale = localeKO
	} else {
		currentLocale = localeEN
	}
}

func GetLocale() string {
	if currentLocale == localeKO {
		return "ko"
	}
	return "en"
}

func T(key MessageKey) string {
	if m, ok := messages[currentLocale][key]; ok {
		return m
	}
	if m, ok := messages[localeEN][key]; ok {
		return m
	}
	return ""
}
