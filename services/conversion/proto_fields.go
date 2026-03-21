package conversion

import (
	"fmt"
	"strings"
	"unicode"

	"google.golang.org/protobuf/reflect/protoreflect"
)

// ProtoFieldStringOrEnumValue reads a protobuf field that may be encoded as a
// string today or as an enum in a future generated API, returning the persisted
// snake_case representation used by the database.
//
//nolint:exhaustive,whitespace // we need only parts
func ProtoFieldStringOrEnumValue(
	msg protoreflect.Message,
	fieldName string,
) (string, error) {
	field := msg.Descriptor().Fields().ByName(protoreflect.Name(fieldName))
	if field == nil {
		return "", fmt.Errorf("field %q not found", fieldName)
	}

	value := msg.Get(field)
	switch field.Kind() {
	case protoreflect.StringKind:
		return value.String(), nil
	case protoreflect.EnumKind:
		enumNumber := value.Enum()
		if enumNumber == 0 {
			return "", nil
		}

		enumValue := field.Enum().Values().ByNumber(enumNumber)
		if enumValue == nil {
			return "", fmt.Errorf("unknown enum value %d for field %q", enumNumber, fieldName)
		}

		persistedValue, ok := persistedValueFromEnumName(
			string(field.Enum().Name()),
			string(enumValue.Name()),
		)
		if !ok {
			return "", fmt.Errorf(
				"unsupported enum value %q for field %q",
				enumValue.Name(),
				fieldName,
			)
		}

		return persistedValue, nil
	default:
		return "", fmt.Errorf("field %q has unsupported kind %s", fieldName, field.Kind())
	}
}

// SetProtoFieldStringOrEnum writes a persisted snake_case value into a protobuf
// field that may be represented as either a string or enum in generated code.
//
//nolint:exhaustive,whitespace,lll // we need only parts
func SetProtoFieldStringOrEnum(
	msg protoreflect.Message,
	fieldName, persistedValue string,
) error {
	field := msg.Descriptor().Fields().ByName(protoreflect.Name(fieldName))
	if field == nil {
		return fmt.Errorf("field %q not found", fieldName)
	}

	switch field.Kind() {
	case protoreflect.StringKind:
		msg.Set(field, protoreflect.ValueOfString(persistedValue))
		return nil
	case protoreflect.EnumKind:
		if persistedValue == "" {
			return nil
		}

		enumNumber, ok := enumNumberForPersistedValue(field.Enum(), persistedValue)
		if ok {
			msg.Set(field, protoreflect.ValueOfEnum(enumNumber))
			return nil
		}

		if unspecified, hasUnspecified := enumNumberForPersistedValue(
			field.Enum(),
			"unspecified",
		); hasUnspecified {
			msg.Set(field, protoreflect.ValueOfEnum(unspecified))
		}

		return fmt.Errorf("unsupported persisted value %q for field %q", persistedValue, fieldName)
	default:
		return fmt.Errorf("field %q has unsupported kind %s", fieldName, field.Kind())
	}
}

//nolint:whitespace // editor/linter issue
func enumNumberForPersistedValue(
	enumDesc protoreflect.EnumDescriptor,
	persistedValue string,
) (protoreflect.EnumNumber, bool) {
	suffix := toScreamingSnake(persistedValue)
	if suffix == "" {
		return 0, false
	}

	prefix := camelToScreamingSnake(string(enumDesc.Name()))
	for _, candidate := range []protoreflect.Name{
		protoreflect.Name(prefix + "_" + suffix),
		protoreflect.Name(suffix),
	} {
		value := enumDesc.Values().ByName(candidate)
		if value != nil {
			return value.Number(), true
		}
	}

	return 0, false
}

func persistedValueFromEnumName(enumName, valueName string) (string, bool) {
	prefix := camelToScreamingSnake(enumName) + "_"
	trimmed := strings.TrimPrefix(valueName, prefix)
	if trimmed == "" || trimmed == "UNSPECIFIED" {
		return "", false
	}

	return strings.ToLower(trimmed), true
}

func camelToScreamingSnake(input string) string {
	if input == "" {
		return ""
	}

	var builder strings.Builder
	runes := []rune(input)
	for idx, r := range runes {
		if idx > 0 && unicode.IsUpper(r) {
			prev := runes[idx-1]
			nextIsLower := idx+1 < len(runes) && unicode.IsLower(runes[idx+1])
			if unicode.IsLower(prev) || nextIsLower {
				builder.WriteByte('_')
			}
		}
		builder.WriteRune(unicode.ToUpper(r))
	}

	return builder.String()
}

func toScreamingSnake(input string) string {
	if input == "" {
		return ""
	}

	var builder strings.Builder
	for _, r := range input {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			builder.WriteRune(unicode.ToUpper(r))
		default:
			builder.WriteByte('_')
		}
	}

	return builder.String()
}
