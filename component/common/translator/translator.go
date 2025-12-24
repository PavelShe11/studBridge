package translator

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/PavelShe11/studbridge/common/domain"
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

		// Validate filename as a language tag
		langTag := strings.TrimSuffix(entry.Name(), ".toml")
		if _, err := language.Parse(langTag); err != nil {
			log.Warnf("skipping file with invalid language tag in filename: %s (%v)", entry.Name(), err)
			continue
		}

		// Load messageFile
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

// TranslateError translates errors that implement TranslatableError interface.
// Supports both BaseError and BaseValidationError types through polymorphism.
func (t *Translator) TranslateError(err error, langs ...string) {
	if err == nil {
		return
	}

	var translatableErr domain.TranslatableError
	ok := errors.As(err, &translatableErr)
	if !ok {
		t.log.Warnf("TranslateError called with non-translatable error type: %T", err)
		return
	}

	localizer := i18n.NewLocalizer(t.bundle, langs...)

	// Pass translate function to the error via polymorphism
	translatableErr.Translate(func(msgID string, params map[string]interface{}) string {
		// If ID is not known, return as-is (might be already translated from another service)
		if _, ok := t.knownIDs[msgID]; !ok {
			return msgID
		}

		translated, err := localizer.Localize(&i18n.LocalizeConfig{
			MessageID:    msgID,
			TemplateData: params,
		})
		if err != nil {
			// Log at debug level - might be template issue, but not critical
			t.log.Debugf("could not translate message '%s': %v", msgID, err)
			return msgID
		}
		return translated
	})
}
