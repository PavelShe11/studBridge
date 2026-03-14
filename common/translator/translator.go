package translator

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/PavelShe11/studbridge/common/entity"
	"github.com/PavelShe11/studbridge/common/logger"

	"github.com/BurntSushi/toml"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

// Translator holds the i18n bundle and the set of known message IDs
type Translator struct {
	bundle   *i18n.Bundle
	knownIDs map[string]struct{}
	log      logger.Logger
}

// NewTranslator creates a new Translator instance
func NewTranslator(log logger.Logger) *Translator {
	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)

	knownIDs := make(map[string]struct{})
	localesDir := "locales"

	entries, err := os.ReadDir(localesDir)
	if err != nil {
		log.Fatalf("failed to read locales directory %s: %v", localesDir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}

		langTag := strings.TrimSuffix(entry.Name(), ".toml")
		if _, err := language.Parse(langTag); err != nil {
			log.Warnf("skipping file with invalid language tag in filename: %s (%v)", entry.Name(), err)
			continue
		}

		filePath := filepath.Join(localesDir, entry.Name())
		messageFile, err := bundle.LoadMessageFile(filePath)
		if err != nil {
			log.Warnf("failed to load message file %s: %v", filePath, err)
			continue
		}

		for _, msg := range messageFile.Messages {
			knownIDs[msg.ID] = struct{}{}
		}
	}

	return &Translator{
		bundle:   bundle,
		knownIDs: knownIDs,
		log:      log,
	}
}

// Translate translates a message by ID with optional template data and language preferences.
// Returns the msgID as-is if the message is not found or translation fails.
func (t *Translator) Translate(msgID string, params map[string]interface{}, langs ...string) string {
	if _, ok := t.knownIDs[msgID]; !ok {
		return msgID
	}

	localizer := i18n.NewLocalizer(t.bundle, langs...)
	translated, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID:    msgID,
		TemplateData: params,
	})
	if err != nil {
		t.log.Debugf("could not translate message '%s': %v", msgID, err)
		return msgID
	}
	return translated
}

// TranslateError translates errors that implement TranslatableError interface.
// Supports both BaseError and BaseValidationError types through polymorphism.
func (t *Translator) TranslateError(err error, langs ...string) {
	if err == nil {
		return
	}

	var translatableErr entity.TranslatableError
	ok := errors.As(err, &translatableErr)
	if !ok {
		t.log.Warnf("TranslateError called with non-translatable error type: %T", err)
		return
	}

	localizer := i18n.NewLocalizer(t.bundle, langs...)

	translatableErr.Translate(func(msgID string, params map[string]interface{}) string {
		if _, ok := t.knownIDs[msgID]; !ok {
			return msgID
		}

		translated, err := localizer.Localize(&i18n.LocalizeConfig{
			MessageID:    msgID,
			TemplateData: params,
		})
		if err != nil {
			t.log.Debugf("could not translate message '%s': %v", msgID, err)
			return msgID
		}
		return translated
	})
}
