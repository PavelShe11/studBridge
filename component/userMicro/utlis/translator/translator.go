package translator

import (
	"github.com/PavelShe11/studbridge/common/logger"
	"github.com/PavelShe11/studbridge/user/internal/domain"
	"os"
	"path/filepath"
	"strings"

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

// TranslateError creates a localizer internally and translates the fields in a domain.BaseError object.
// Supports parameterized translations through TemplateData from error.FieldErrors[].Params
func (t *Translator) TranslateError(err *domain.BaseError, langs ...string) {
	if err == nil {
		return
	}

	localizer := i18n.NewLocalizer(t.bundle, langs...)

	// Translate the top-level error name
	if _, ok := t.knownIDs[err.Name]; ok {
		translatedName, localizeErr := localizer.Localize(&i18n.LocalizeConfig{
			MessageID: err.Name,
		})
		if localizeErr == nil {
			err.Name = translatedName
		} else {
			t.log.Warnf("translation failed for top-level error name: %s", err.Name)
		}
	}

	// Translate field errors with parameter support
	for i := range err.FieldErrors {
		msgID := err.FieldErrors[i].Message
		if _, ok := t.knownIDs[msgID]; ok {
			// Build template data from params
			templateData := make(map[string]interface{})
			for k, v := range err.FieldErrors[i].Params {
				templateData[k] = v
			}

			translatedMsg, localizeErr := localizer.Localize(&i18n.LocalizeConfig{
				MessageID:    msgID,
				TemplateData: templateData,
			})
			if localizeErr == nil {
				err.FieldErrors[i].Message = translatedMsg
			} else {
				t.log.Warnf("translation failed for field error message: %s", msgID)
			}
		}
	}
}
