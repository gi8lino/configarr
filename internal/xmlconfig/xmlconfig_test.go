package xmlconfig

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocumentUpdates(t *testing.T) {
	tests := []struct {
		name      string
		source    string
		path      string
		value     string
		want      string
		wantError bool
	}{
		{name: "BOM", source: "\xef\xbb\xbf<Root><Key>old</Key></Root>", path: "/Root/Key", value: "new", want: "\xef\xbb\xbf<Root><Key>new</Key></Root>"},
		{name: "duplicate attributes", source: `<Root key="a" key="b"/>`, path: "/Root/@key", wantError: true},
		{name: "nested text", source: `<Root><Database><Host>old</Host></Database></Root>`, path: "/Root/Database/Host", value: "new", want: `<Root><Database><Host>new</Host></Database></Root>`},
		{name: "quoted attribute", source: `<Root host = 'old' other="unchanged"/>`, path: "/Root/@host", value: `x'"&<`, want: `<Root host = 'x&#39;&#34;&amp;&lt;' other="unchanged"/>`},
		{name: "self closing", source: `<Root><Key /></Root>`, path: "/Root/Key", value: "new", want: `<Root><Key >new</Key></Root>`},
		{name: "empty element", source: `<Root><Key></Key></Root>`, path: "/Root/Key", value: "new", want: `<Root><Key>new</Key></Root>`},
		{name: "empty replacement", source: `<Root><Key>old</Key></Root>`, path: "/Root/Key", want: `<Root><Key></Key></Root>`},
		{name: "escaped text", source: `<Root><Key>old</Key></Root>`, path: "/Root/Key", value: "a<&", want: `<Root><Key>a&lt;&amp;</Key></Root>`},
		{name: "comment and CDATA", source: `<?xml version="1.0"?><Root><!--keep--><Key><![CDATA[old]]><!--inner--> text</Key></Root>`, path: "/Root/Key", value: "new", want: `<?xml version="1.0"?><Root><!--keep--><Key>new<!--inner--></Key></Root>`},
		{name: "semantic no op", source: `<Root><Key><![CDATA[a&b]]></Key></Root>`, path: "/Root/Key", value: "a&b", want: `<Root><Key><![CDATA[a&b]]></Key></Root>`},
		{name: "positional", source: `<Root><Key>a</Key><Key>b</Key></Root>`, path: "/Root/Key[2]", value: "c", want: `<Root><Key>a</Key><Key>c</Key></Root>`},
		{name: "attribute predicate", source: `<Root><Key name="other">a</Key><Key name="a/b">b</Key></Root>`, path: `/Root/Key[@name='a/b']`, value: "c", want: `<Root><Key name="other">a</Key><Key name="a/b">c</Key></Root>`},
		{name: "qualified names", source: `<p:Root xmlns:p="urn:r" xmlns:q="urn:a"><p:Key q:name="old">v</p:Key></p:Root>`, path: "/p:Root/p:Key/@q:name", value: "new", want: `<p:Root xmlns:p="urn:r" xmlns:q="urn:a"><p:Key q:name="new">v</p:Key></p:Root>`},
		{name: "default namespace", source: `<Root xmlns="urn:r"><Key>old</Key></Root>`, path: "/Root/Key", value: "new", want: `<Root xmlns="urn:r"><Key>new</Key></Root>`},
		{name: "ambiguous", source: `<Root><Key>a</Key><Key>b</Key></Root>`, path: "/Root/Key", wantError: true},
		{name: "missing", source: `<Root/>`, path: "/Root/Key", wantError: true},
		{name: "missing attribute", source: `<Root/>`, path: "/Root/@host", wantError: true},
		{name: "non leaf", source: `<Root><Key>v</Key></Root>`, path: "/Root", wantError: true},
		{name: "invalid text", source: `<Root/>`, path: "/Root", value: "bad\x00text", wantError: true},
		{name: "multiple roots", source: `<Root/><Root/>`, path: "/Root", wantError: true},
		{name: "malformed", source: `<Root><Key></Root>`, path: "/Root/Key", wantError: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			update := Update{ID: "test", Path: tc.path, Value: tc.value}

			got, err := applyUpdate([]byte(tc.source), update)
			if tc.wantError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			assert.Equal(t, tc.want, string(got))
			_, err = planEdits(bytes.NewReader(got), int64(len(got)), nil)
			assert.NoError(t, err, "updated XML must remain valid")
		})
	}
}

func TestRejectUnsupportedSelectors(t *testing.T) {
	paths := []string{
		"", "Root/Key", "/", "//Key", "/Root/", "/Root/Key[0]", "/Root/Key[-1]",
		"/Root/Key[", "/Root/Key]", "/Root/Key[@name=bad]", "/Root/Key[@name='bad]",
		"/Root/Key[1][2]", "/Root/Key[[1]]", "/@host", "/Root/@", "/Root/Key|/Root/Other",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			update := Update{ID: "test", Path: path, Value: "new"}

			_, err := applyUpdate([]byte(`<Root><Key>old</Key></Root>`), update)

			assert.Error(t, err)
		})
	}
}

func TestRejectTrailingDocumentContent(t *testing.T) {
	for _, suffix := range []string{`<Config/>`, `garbage`, `<broken`} {
		t.Run(suffix, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.xml")
			source := []byte(`<Config><Key>value</Key></Config>` + suffix)
			require.NoError(t, os.WriteFile(path, source, 0600))

			_, err := EditFile(path, nil)

			assert.Error(t, err)
		})
	}
}

func FuzzDocumentUpdates(f *testing.F) {
	for _, source := range []string{
		`<Root/>`, `<Root key='value'/>`, `<Root>old<!--keep--><![CDATA[text]]></Root>`,
		`<p:Root xmlns:p="urn:test" p:key="value"/>`, `<Root><Child/></Root>`, `<Root`,
	} {
		f.Add(source, "replacement<&")
	}

	f.Fuzz(func(t *testing.T, source, value string) {
		_, err := planEdits(strings.NewReader(source), int64(len(source)), nil)
		if err != nil {
			return
		}

		result, err := applyUpdate([]byte(source), Update{ID: "fuzz", Path: "/*", Value: value})
		if err != nil {
			return
		}

		_, err = planEdits(bytes.NewReader(result), int64(len(result)), nil)
		require.NoError(t, err, "update produced invalid XML")
	})
}

func applyUpdate(source []byte, update Update) ([]byte, error) {
	reader := bytes.NewReader(source)
	edits, err := planEdits(reader, int64(len(source)), []Update{update})
	if err != nil {
		return nil, err
	}

	var result bytes.Buffer
	if err := copyEdits(reader, int64(len(source)), &result, edits); err != nil {
		return nil, err
	}

	return result.Bytes(), nil
}
