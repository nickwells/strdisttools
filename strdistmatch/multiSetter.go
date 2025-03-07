package main

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/nickwells/english.mod/english"
	"github.com/nickwells/param.mod/v6/param"
	"github.com/nickwells/param.mod/v6/psetter"
	"golang.org/x/exp/maps"
)

// the indexes for the parts returned by FindAllStringSubmatch for the
// entryValRE
const (
	wholeValIdx = iota
	keyIdx
	valIdx
	expectedRegexpSubmatchLength
)

const maxAltNames = 3

// The regular expression fragments used to build the subValueRE
const (
	keyRE = `[_a-zA-Z][_a-zA-Z0-9]*`
	eqRE  = `\s*=\s*`
	strRE = `[^"]*`
)

var (
	evsKeyRE = regexp.MustCompile(keyRE)

	subValueRE = regexp.MustCompile(
		`\s*` + // match & skip any space at the start
			`(` + keyRE + `)` + // match & keep the key
			eqRE + // match & skip the '=' (& any surrounding space)
			`"(` + strRE + `)"` + // match & keep the string content only
			`\s*`) // match and skip any space at the end
)

const multiSetterValueForm = `name=subval="..." subval="..." ...`

// MultiSetterActionFunc is the type of a function that can be supplied to be
// run after the value has been successfully changed
type MultiSetterActionFunc func(entryValName string, entryValValue string) error

// EntryValSetter holds the configuration for the entry value setters
type EntryValSetter struct {
	// Setter is the setter that will be called.
	Setter param.Setter
	// PostActionFuncs is a, possibly empty, list of functions to be called
	// after the Setter has successfully completed
	PostActionFuncs []MultiSetterActionFunc
	// MustBeSet will force an error to be generated if this Setter is not
	// called for a MultiSetter value.
	MustBeSet bool
}

// NamedValue associates a name with a value. It is used as the entry type in
// a slice of values when the MultiSetter is populating a slice rather than a
// map. The MultiSetter associates a name (of type S) with a value (of type
// T).
//
// If you want there to be only one value for each name then you can make
// the association using a map.
//
// If you want to have multiple, different values for a given name then you
// can make the association with a slice of NamedValue's.
type NamedValue[S ~string, T any] struct {
	Name  S
	Value T
}

// MapMultiSetter allows multiple values to be set in a map entry with a
// single parameter. The complexity in the setup is mostly in the setting of
// the MultiSetter embedded type; see the documentation for that type for
// details on how to initialise it.
//
// For the rest you must firstly, as usual, set the Value pointer to the
// value you want to set. The value must be a map from your string type to
// your data type
//
// Then populate the MultiSetter.
type MapMultiSetter[S ~string, T any] struct {
	psetter.ValueReqMandatory

	// Value must be set, the program will panic if not. This is the map of
	// values that this setter is setting
	Value *map[S]T
	// AllowHiddenMapEntries lets you have a Value which has an existing
	// entry whose key is not in the MultiSetter's AVals map. Normally a
	// Value having an illegal key would cause a panic from CheckSetter but
	// this allows such entries. Note that this has no effect if the AVals
	// map is empty. Note also that any entry with a disallowed key cannot be
	// changed through this param.Setter.
	AllowHiddenMapEntries bool

	// MultiSetter does the heavy lifting for this Setter type. It provides
	// the bulk of the code that implements the Setter interface.
	MultiSetterBase[S, T]
}

// SetWithVal populates the Value map with the parameters given by the
// paramVal. If any error is reported the Value is left unchanged.
func (s *MapMultiSetter[S, T]) SetWithVal(_, paramVal string) error {
	nv, err := s.GetNamedValue("", paramVal)
	if err != nil {
		return err
	}

	(*s.Value)[nv.Name] = nv.Value

	return nil
}

// CurrentValue returns the current setting of the parameter value
func (s MapMultiSetter[S, T]) CurrentValue() string {
	valueParts := []string{}
	for k, v := range *s.Value {
		valueParts = append(valueParts, fmt.Sprintf("%q: %#v", k, v))
	}

	return strings.Join(valueParts, " ")
}

// CheckSetter panics if the setter has not been properly created - if the
// Value is nil, if there are no EntryValSetters
func (s MapMultiSetter[S, T]) CheckSetter(name string) {
	intro := name + ": MultiSetterMap Check failed: "

	s.checkValue(intro)
	s.CheckMultiSetter(intro)
}

// checkValue checks the Value and panics if it is invalid. Note that it also
// sets the map to a non-nil value if the pointed to map is nil.
func (s MapMultiSetter[S, T]) checkValue(intro string) {
	if s.Value == nil {
		panic(intro + "the Value to be set is nil")
	}

	if *s.Value == nil {
		*s.Value = map[S]T{}
	}

	if len(s.AVals) == 0 ||
		s.AllowHiddenMapEntries {
		return
	}

	for k := range *s.Value {
		if !s.AVals.ValueAllowed(string(k)) {
			panic(fmt.Sprintf("%sthe map entry with key %q is invalid"+
				" - it is not in the allowed values map",
				intro, k))
		}
	}
}

// ListMultiSetter allows multiple values to be set in a map entry with a
// single parameter. The complexity in the setup is mostly in the setting of
// the MultiSetter embedded type; see the documentation for that type for
// details on how to initialise it.
//
// For the rest you must firstly, as usual, set the Value pointer to the
// value you want to set. The value must be a map from your string type to
// your data type
//
// Then populate the MultiSetter.
type ListMultiSetter[S ~string, T any] struct {
	psetter.ValueReqMandatory

	// Value must be set, the program will panic if not. This is the map of
	// values that this setter is setting
	Value *[]NamedValue[S, T]
	// AllowInvalidListEntries lets you have a Value which has an existing
	// entry whose key is not in the MultiSetter's AVals map. Normally a
	// Value having an illegal key would cause a panic from CheckSetter but
	// this allows such entries. Note that this has no effect if the AVals
	// map is empty. Note also that any entry with a disallowed key cannot be
	// changed through this param.Setter.
	AllowInvalidListEntries bool

	// MultiSetterBase does the heavy lifting for this Setter type. It provides
	// the bulk of the code that implements the Setter interface.
	MultiSetterBase[S, T]
}

// SetWithVal populates the Value map with the parameters given by the
// paramVal. If any error is reported the Value is left unchanged.
func (s *ListMultiSetter[S, T]) SetWithVal(_, paramVal string) error {
	nv, err := s.GetNamedValue("", paramVal)
	if err != nil {
		return err
	}

	(*s.Value) = append((*s.Value), nv)

	return nil
}

// CurrentValue returns the current setting of the parameter value
func (s ListMultiSetter[S, T]) CurrentValue() string {
	valueParts := []string{}
	for _, nv := range *s.Value {
		valueParts = append(valueParts,
			fmt.Sprintf("%q: %#v", nv.Name, nv.Value))
	}

	return strings.Join(valueParts, " ")
}

// CheckSetter panics if the setter has not been properly created - if the
// Value is nil, if there are no EntryValSetters
func (s ListMultiSetter[S, T]) CheckSetter(name string) {
	intro := name + ": MultiSetterList Check failed: "

	s.checkValue(intro)
	s.CheckMultiSetter(intro)
}

// checkValue checks the Value and panics if it is invalid. Note that it also
// sets the map to a non-nil value if the pointed to map is nil.
func (s ListMultiSetter[S, T]) checkValue(intro string) {
	if s.Value == nil {
		panic(intro + "the Value to be set is nil")
	}

	if len(s.AVals) == 0 ||
		s.AllowInvalidListEntries {
		return
	}

	for _, nv := range *s.Value {
		if !s.AVals.ValueAllowed(string(nv.Name)) {
			panic(fmt.Sprintf("%sthe map entry with key %q is invalid"+
				" - it is not in the allowed values map",
				intro, nv.Name))
		}
	}
}

// MultiSetterBase is the engine used by the ...MultiSetter types to
// construct the named collection of values. It allows multiple values to be
// set with a single parameter. It is a bit complicated to set up as it is
// self-referential so there is a little more explanation than with most
// param.Setters.
//
// Firstly, you can choose to set the DfltEntryVal to some value but if you
// are happy with the zero values there is no need to do this. Whatever value
// you give here will be copied into the EntryVal before setting the
// EntryVal from the parameter value.
//
// Then you must construct the collection of EntryValSetterMap. Each
// param.Setter here is called when a sub-string matches its name in the
// EntryValSetterMap. The values that these param.Setters refer to should all
// be members of the EntryVal. Also each param.Setter must be one that takes
// a param value.
//
// The different ...MultiSetter types each have their own SetWithVal methods
// which call the MultiSetterBase's GetNamedValue and use the results to
// populate their own internal Value element. The GetNamedValue will copy the
// DfltEntryVal over the EntryVal, call the EntryValSetters according to the
// parameter value and then return a populated NamedValue which the
// ...MultiSetter can use to populate its own Value member. If any errors are
// detected then an empty NamedValue is returned.
type MultiSetterBase[S ~string, T any] struct {
	// DfltEntryVal holds the default values to give the entries in the Value
	// map. If the zero values are OK there is no need to change this when
	// creating the MultiSetter.
	DfltEntryVal T
	// EntryVal is used purely as a target for the EntryValSetters. Its value
	// is overwritten each time the SetWithVal method is called when it is
	// initialised to the DfltEntryVal
	EntryVal T
	// EntryValSetterMap must be set, the program will panic if not. Each
	// param.Setter should have a Value that refers to a member of the
	// MultiSetter.EntryVal. Also only setters that expect a value are
	// allowed. The 'subval' names refer to entries in this map.
	EntryValSetterMap map[string]EntryValSetter

	// AVals need not be set but if it has any entries then they will be used
	// to constrain the allowed 'name' part (note not the subval name part)
	// of the value being set.
	AVals psetter.AllowedVals[S]

	// EntryValSMAliases need not be set but if it has any entries then the
	// key must not appear in the EntryValSetterMap and the mapped value must
	// appear.
	EntryValSMAliases map[string]string
}

// GetNamedValue (called when a value follows the parameter) populates an entry
// in Value map with the 'name' taken from the first part of the string
// (before the '=', if any) and the 'subval' parts, if any, taken from the
// parts after the '='. If the AllowedVals are not empty then the name must
// be an allowed value.
//
// Note that, unusually, it takes a pointer receiver so a pointer to a
// MultiSetter must be given to satisfy the param.Setter interface.
func (s *MultiSetterBase[S, T]) GetNamedValue(_ string, paramVal string) (
	NamedValue[S, T], error,
) {
	name, val, ok := strings.Cut(paramVal, "=")

	err := s.checkParamPartName(name)
	if err != nil {
		return NamedValue[S, T]{}, err
	}

	if !ok {
		return NamedValue[S, T]{Name: S(name), Value: s.DfltEntryVal}, nil
	}

	s.EntryVal = s.DfltEntryVal

	subValues := subValueRE.FindAllStringSubmatch(val, -1)
	if len(subValues) == 0 {
		return NamedValue[S, T]{},
			fmt.Errorf("cannot get any values from the parameter: %q", val)
	}

	dups := map[string]string{}

	for i, sVal := range subValues {
		val, err = s.setWithSubval(val, i, sVal, dups)
		if err != nil {
			return NamedValue[S, T]{}, err
		}
	}

	for evKey, evs := range s.EntryValSetterMap {
		if evs.MustBeSet {
			if _, ok := dups[evKey]; !ok {
				return NamedValue[S, T]{},
					fmt.Errorf(
						"the subvalue for %q must be set but hasn't been",
						evKey)
			}
		}
	}

	// if any of the text is left after parsing, that is an error.
	if val != "" {
		return NamedValue[S, T]{},
			fmt.Errorf(
				"unexpected text: %q, at the end of the parameter value",
				val)
	}

	// All's well, set the value.
	return NamedValue[S, T]{Name: S(name), Value: s.EntryVal}, nil
}

// checkParamPartName checks that the name part of the parameter value (the
// part before the first '=') is valid and it will return an error if not.
func (s MultiSetterBase[S, T]) checkParamPartName(name string) error {
	if name == "" {
		return errors.New("the name may not be empty")
	}

	if len(s.AVals) > 0 {
		if !s.AVals.ValueAllowed(name) {
			pop := []string{}
			for k := range s.AVals {
				pop = append(pop, string(k))
			}

			return fmt.Errorf(
				"bad name: %q, the name is not recognised%s",
				name, SuggestAlternatives(maxAltNames, name, pop))
		}
	}

	return nil
}

// setWithSubval sets a field in the EntryVal by calling the appropriate
// param.Setter. If any error is detected it is returned.
func (s *MultiSetterBase[S, T]) getSetter(i int, evKey string,
) (string, EntryValSetter, error) {
	if evs, ok := s.EntryValSetterMap[evKey]; ok {
		return evKey, evs, nil
	}

	if aliasKey, ok := s.EntryValSMAliases[evKey]; ok {
		if evs, ok := s.EntryValSetterMap[aliasKey]; ok {
			return aliasKey, evs, nil
		}
	}

	entryValNames := maps.Keys(s.EntryValSetterMap)
	aliasNames := maps.Keys(s.EntryValSMAliases)
	entryValNames = append(entryValNames, aliasNames...)

	return evKey, EntryValSetter{},
		fmt.Errorf("bad sub-value name (%q), at the %d%s entry%s",
			evKey, i+1, english.OrdinalSuffix(i+1),
			SuggestAlternatives(maxAltNames, evKey, entryValNames))
}

// setWithSubval sets a field in the EntryVal by calling the appropriate
// param.Setter. If any error is detected it is returned.
func (s *MultiSetterBase[S, T]) setWithSubval(
	wholeValue string,
	i int,
	sVal []string,
	dups map[string]string,
) (string, error) {
	const expectedVal = `expecting name="string"`

	// First check that the parts of the subval are all present
	if len(sVal) != expectedRegexpSubmatchLength {
		return wholeValue,
			fmt.Errorf("cannot parse the %d%s sub-value, %s",
				i+1, english.OrdinalSuffix(i+1), expectedVal)
	}

	// Now check that there is no text in the value before matched
	// subval="..." part. This catches "syntax" errors in the value string.
	wholeSubValue := sVal[wholeValIdx]

	if !strings.HasPrefix(wholeValue, wholeSubValue) {
		badVal, _, _ := strings.Cut(wholeValue, wholeSubValue)

		return wholeValue,
			fmt.Errorf("unexpected text: %q, before the %d%s entry: %q, %s",
				badVal,
				i+1, english.OrdinalSuffix(i+1),
				wholeSubValue, expectedVal)
	}

	wholeValue = strings.TrimPrefix(wholeValue, wholeSubValue)

	// Now get the param.Setter for this subval
	evKey, evVal := sVal[keyIdx], sVal[valIdx]

	evKey, evs, err := s.getSetter(i, evKey)
	if err != nil {
		return wholeValue, err
	}

	// Now check that we haven't seen this subval key before
	prevVal, dupFound := dups[evKey]
	if dupFound {
		return wholeValue, fmt.Errorf(
			"the value for %q has been set twice,"+
				" with %q and then with %q (the %d%s entry)",
			evKey, prevVal, wholeSubValue,
			i+1, english.OrdinalSuffix(i+1))
	}

	dups[evKey] = wholeSubValue

	// Lastly run the param.Setter's SetWithVal method
	if err := evs.Setter.SetWithVal(evKey, evVal); err != nil {
		return wholeValue, err
	}

	for _, f := range evs.PostActionFuncs {
		if err := f(evKey, evVal); err != nil {
			return wholeValue, err
		}
	}

	return wholeValue, nil
}

// allowedValuesNames returns a string reflecting the allowed name
// values. Note that this may be empty if the AVals map is empty.
func (s MultiSetterBase[S, T]) allowedValuesNames() string {
	if len(s.AVals) == 0 {
		return ""
	}

	str := "\n\n" +
		"the allowed names are"
	names, maxLen := s.AVals.Keys()

	sort.Strings(names)

	for _, n := range names {
		str += fmt.Sprintf("\n- %-*s: %s",
			maxLen, n, s.AVals[S(n)])
	}

	return str
}

// allowedValuesSubvals returns a string reflecting the subval names and
// allowed values
func (s MultiSetterBase[S, T]) allowedValuesSubvals() string {
	str := "\n\n" +
		"the allowed"
	if len(s.EntryValSetterMap) > 1 {
		str += " subval names and values are:"
	} else {
		str += " subval name and value is:"
	}

	maxLen := 0
	evsKeys := []string{}

	for k := range s.EntryValSetterMap {
		evsKeys = append(evsKeys, k)

		if len(k) > maxLen {
			maxLen = len(k)
		}
	}

	sort.Strings(evsKeys)

	valSeenBefore := map[string]string{}

	for _, k := range evsKeys {
		val := s.EntryValSetterMap[k].Setter.AllowedValues()
		if prevKey, ok := valSeenBefore[val]; ok {
			val = `as for "` + prevKey + `"`
		} else {
			valSeenBefore[val] = k
		}

		str += fmt.Sprintf("\n- %-*s: %s", maxLen, k, val)
	}

	return str
}

// allowedValuesSubvalAliases returns a string given any alias names
// allowed. Note that this can be empty if the Aliases map is empty.
func (s MultiSetterBase[S, T]) allowedValuesSubvalAliases() string {
	if len(s.EntryValSMAliases) == 0 {
		return ""
	}

	str := "\n\n"

	if len(s.EntryValSMAliases) == 1 {
		str += "the following alias for the subval name is allowed: "
	} else {
		str += "the following aliases for the subval names are allowed: "
	}

	maxLen := 0
	aliases := []string{}

	for k := range s.EntryValSMAliases {
		aliases = append(aliases, k)

		if len(k) > maxLen {
			maxLen = len(k)
		}
	}

	sort.Strings(aliases)

	for _, k := range aliases {
		str += fmt.Sprintf("\n- %-*s: %s", maxLen, k, s.EntryValSMAliases[k])
	}

	return str
}

// AllowedValues returns a string describing the allowed values
func (s MultiSetterBase[S, T]) AllowedValues() string {
	avStr := "a value of the form " + multiSetterValueForm

	avStr += s.allowedValuesNames()

	avStr += s.allowedValuesSubvals()

	avStr += s.allowedValuesSubvalAliases()

	return avStr
}

// ValDescribe returns a short string illustrating the value to be supplied
func (s MultiSetterBase[S, T]) ValDescribe() string {
	return multiSetterValueForm
}

// CheckMultiSetter panics if the multi-setter has not been properly created
func (s MultiSetterBase[S, T]) CheckMultiSetter(intro string) {
	s.checkAllowedValues(intro)
	s.checkEntryValSetters(intro)
	s.checkEntryValSetterMapAliases(intro)
}

// checkAllowedValues checks the AVals and panics if there are any
// problems. It also checks that any existing entries in the Value map have
// keys in the AVals map.
func (s MultiSetterBase[S, T]) checkAllowedValues(intro string) {
	if len(s.AVals) == 0 {
		return
	}

	if err := s.AVals.Check(); err != nil {
		panic(intro + err.Error())
	}
}

// checkEntryValSetters checks the EntryValSetters and panics if it is
// invalid. It checks that the keys (the names of the sub-entries) match the
// regular expression, that the setters all take a value and that the
// individual setters themselves pass their own checks.
func (s MultiSetterBase[S, T]) checkEntryValSetters(intro string) {
	if len(s.EntryValSetterMap) == 0 {
		panic(intro + "there must be at least one sub-value setter")
	}

	for k, evs := range s.EntryValSetterMap {
		evsIntro := fmt.Sprintf("%sbad entry-value setter: %q: ", intro, k)

		if !evsKeyRE.MatchString(k) {
			panic(fmt.Sprintf(
				"%sbad key %q: it should be"+
					" a letter followed by zero or more letters or numbers",
				evsIntro, k))
		}

		if evs.Setter.ValueReq() == param.None {
			panic(fmt.Sprintf("%sit must take a value", evsIntro))
		}

		evs.Setter.CheckSetter(intro + ".SubTypeSetters[" + k + "]")
	}
}

// checkEntryValSetterMapAliases checks the EntryValSMAliases and panics if
// there are any problems.
func (s MultiSetterBase[S, T]) checkEntryValSetterMapAliases(intro string) {
	for k, v := range s.EntryValSMAliases {
		if _, ok := s.EntryValSetterMap[k]; ok {
			panic(fmt.Sprintf(
				"%sthe alias %q is the same as a subval name", intro, k))
		}

		if _, ok := s.EntryValSetterMap[v]; !ok {
			panic(fmt.Sprintf(
				"%sthe alias %q (= %q) does not refer to a subval name",
				intro, k, v))
		}
	}
}
