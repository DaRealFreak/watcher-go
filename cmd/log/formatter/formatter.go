package formatter

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
	"github.com/mgutz/ansi"
	"github.com/sirupsen/logrus"
)

// defaultTimestampFormat is the time format we use if nothing is set manually
const defaultTimestampFormat = time.StampMilli

// nolint: gochecknoglobals
var (
	FieldMatchColorScheme map[string][]*FieldMatch
	defaultColorSchema    = &ColorSchema{
		Timestamp:  "black+h",
		InfoLevel:  "green",
		WarnLevel:  "yellow+B",
		ErrorLevel: "red",
		FatalLevel: "red",
		PanicLevel: "red",
		DebugLevel: "blue",
	}
)

// FieldMatch contains the value and defined color of the field match
type FieldMatch struct {
	Value interface{}
	Color string
}

// ColorSchema is the color schema for the default log parts/levels
type ColorSchema struct {
	Timestamp  string
	InfoLevel  string
	WarnLevel  string
	ErrorLevel string
	FatalLevel string
	PanicLevel string
	DebugLevel string
}

// Formatter contains all options for this formatter
type Formatter struct {
	// no colors
	DisableColors bool
	// no check for TTY terminal
	ForceColors bool
	// no check for TTY terminal
	ForceFormatting bool
	// no timestamp
	DisableTimestamp bool
	// false -> info, true -> INFO
	DisableLowercase bool
	// false -> time passed, true -> timestamp
	FullTimestamp bool
	// timestamp formatting, default is time.RFC3339
	TimestampFormat string
	// color schema for messages
	ColorSchema      *ColorSchema
	PadAllLogEntries bool
}

// Format implements the interface method of the logrus Formatter
func (f *Formatter) Format(entry *logrus.Entry) ([]byte, error) {
	out := new(bytes.Buffer)

	// if colors are not disabled and no color schema got set we use the default color schema
	if f.ColorSchema == nil {
		f.ColorSchema = defaultColorSchema
	}

	timestampFormat := f.TimestampFormat
	if timestampFormat == "" {
		timestampFormat = defaultTimestampFormat
	}

	colorSchema := f.getEntryColor(entry, f.ColorSchema.Timestamp, defaultColorSchema.Timestamp)
	if _, err := out.WriteString(colorSchema(entry.Time.Format(timestampFormat))); err != nil {
		return nil, err
	}

	if err := f.addPadding(out); err != nil {
		return nil, err
	}
	colorSchema = f.getLevelColor(entry)
	if _, err := out.WriteString(colorSchema(fmt.Sprintf("%7s", entry.Level.String()))); err != nil {
		return nil, err
	}

	if err := f.appendPrependedFields(out, entry); err != nil {
		return nil, err
	}

	if err := f.addPadding(out); err != nil {
		return nil, err
	}
	if _, err := out.Write([]byte(entry.Message)); err != nil {
		return nil, err
	}

	for fieldKey, fieldValue := range entry.Data {
		if err := f.addPadding(out); err != nil {
			return nil, err
		}
		if _, err := out.Write([]byte(fmt.Sprintf("%s=%v", colorSchema(fieldKey), fieldValue))); err != nil {
			return nil, err
		}
	}

	if _, err := out.Write([]byte("\n")); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// appendPrependedFields appends the prepended fields and removes them from the data attribute in the entry
func (f *Formatter) appendPrependedFields(out io.Writer, entry *logrus.Entry) (err error) {
	for fieldKey, fieldMatches := range FieldMatchColorScheme {
		// check for the longest value for the required padding on PadAllLogEntries = true
		longestValue := 0
		if f.PadAllLogEntries {
			for _, fieldMatch := range FieldMatchColorScheme[fieldKey] {
				if longestValue < len(fmt.Sprintf("[%v]", fieldMatch.Value)) {
					longestValue = len(fmt.Sprintf("[%v]", fieldMatch.Value))
				}
			}
		}
		padded := false
		// use the longest value for the padding (always 0 if PadAllLogEntries = false)
		outFormat := fmt.Sprintf("%%%ds", longestValue)
		for entryKey, entryValue := range entry.Data {
			for _, matchValue := range fieldMatches {
				if entryKey == fieldKey && entryValue == matchValue.Value {
					// match found, write the (colored) value into the passed writer
					colorSchema := f.getEntryColor(entry, matchValue.Color, "")
					_, err = out.Write(
						[]byte(" " + colorSchema(fmt.Sprintf(outFormat, fmt.Sprintf("[%v]", entryValue)))),
					)
					if err != nil {
						return err
					}
					delete(entry.Data, entryKey)
					// prevent double padding which is intended for no matches
					padded = true
					break
				}
			}
		}
		// add padding if no match got found and PadAllLogEntries is enabled
		if f.PadAllLogEntries && !padded {
			if err := f.addPadding(out); err != nil {
				return err
			}
			_, err = out.Write([]byte(fmt.Sprintf(outFormat, "")))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// isTerminal checks if we are currently in a terminal
func (f *Formatter) isTerminal(writer io.Writer) bool {
	// check the type since the file descriptor is only callable for files, so we can't access it directly
	switch out := writer.(type) {
	case *os.File:
		return isatty.IsCygwinTerminal(out.Fd())
	case *colorable.Writer:
		return isatty.IsTerminal(os.Stdout.Fd())
	default:
		return false
	}
}

// getLevelColor returns the ansi ColorFunc depending on the log entry level
func (f *Formatter) getLevelColor(entry *logrus.Entry) func(string) string {
	// disabled colors or not a terminal
	if f.DisableColors || (!f.isTerminal(entry.Logger.Out) && !f.ForceColors) {
		return ansi.ColorFunc("")
	}
	switch entry.Level {
	case logrus.InfoLevel:
		return f.getEntryColor(entry, f.ColorSchema.InfoLevel, defaultColorSchema.InfoLevel)
	case logrus.WarnLevel:
		return f.getEntryColor(entry, f.ColorSchema.WarnLevel, defaultColorSchema.WarnLevel)
	case logrus.ErrorLevel:
		return f.getEntryColor(entry, f.ColorSchema.ErrorLevel, defaultColorSchema.ErrorLevel)
	case logrus.FatalLevel:
		return f.getEntryColor(entry, f.ColorSchema.FatalLevel, defaultColorSchema.FatalLevel)
	case logrus.PanicLevel:
		return f.getEntryColor(entry, f.ColorSchema.PanicLevel, defaultColorSchema.PanicLevel)
	case logrus.DebugLevel:
		return f.getEntryColor(entry, f.ColorSchema.DebugLevel, defaultColorSchema.DebugLevel)
	}
	return ansi.ColorFunc("")
}

// getEntryColor checks if we have a terminal and colors are not disabled and returns the ansi ColorFunc
func (f *Formatter) getEntryColor(entry *logrus.Entry, customColor string, defaultColor string) func(string) string {
	// disabled colors or not a terminal
	if f.DisableColors || (!f.isTerminal(entry.Logger.Out) && !f.ForceColors) {
		return ansi.ColorFunc("")
	}
	style := defaultColor
	if customColor != "" {
		style = customColor
	}
	return ansi.ColorFunc(style)
}

// addPadding adds the assigned padding character and writes it to our buffer
func (f *Formatter) addPadding(writer io.Writer) (err error) {
	_, err = writer.Write([]byte(" "))
	return err
}

// AddFieldMatchColorScheme registers field match color scheme
func AddFieldMatchColorScheme(key string, match *FieldMatch) {
	if FieldMatchColorScheme == nil {
		FieldMatchColorScheme = make(map[string][]*FieldMatch)
	}
	FieldMatchColorScheme[key] = append(FieldMatchColorScheme[key], match)
}
