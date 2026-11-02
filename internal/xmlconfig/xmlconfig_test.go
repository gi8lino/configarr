package xmlconfig

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanEdits(t *testing.T) {
	t.Parallel()

	t.Run("BOM", func(t *testing.T) {
		t.Parallel()

		update := Update{ID: "test", Path: "/Root/Key", Value: "new"}

		got, err := applyUpdate([]byte("\xef\xbb\xbf<Root><Key>old</Key></Root>"), update)
		require.NoError(t, err)

		assert.Equal(t, "\xef\xbb\xbf<Root><Key>new</Key></Root>", string(got))
		_, err = planEdits(bytes.NewReader(got), int64(len(got)), nil)
		assert.NoError(t, err, "updated XML must remain valid")
	})

	t.Run("duplicate attributes", func(t *testing.T) {
		t.Parallel()

		update := Update{ID: "test", Path: "/Root/@key", Value: ""}

		_, err := applyUpdate([]byte(`<Root key="a" key="b"/>`), update)
		assert.Error(t, err)
	})

	t.Run("nested text", func(t *testing.T) {
		t.Parallel()

		update := Update{ID: "test", Path: "/Root/Database/Host", Value: "new"}

		got, err := applyUpdate([]byte(`<Root><Database><Host>old</Host></Database></Root>`), update)
		require.NoError(t, err)

		assert.Equal(t, `<Root><Database><Host>new</Host></Database></Root>`, string(got))
		_, err = planEdits(bytes.NewReader(got), int64(len(got)), nil)
		assert.NoError(t, err, "updated XML must remain valid")
	})

	t.Run("quoted attribute", func(t *testing.T) {
		t.Parallel()

		update := Update{ID: "test", Path: "/Root/@host", Value: `x'"&<`}

		got, err := applyUpdate([]byte(`<Root host = 'old' other="unchanged"/>`), update)
		require.NoError(t, err)

		assert.Equal(t, `<Root host = 'x&#39;&#34;&amp;&lt;' other="unchanged"/>`, string(got))
		_, err = planEdits(bytes.NewReader(got), int64(len(got)), nil)
		assert.NoError(t, err, "updated XML must remain valid")
	})

	t.Run("self closing", func(t *testing.T) {
		t.Parallel()

		update := Update{ID: "test", Path: "/Root/Key", Value: "new"}

		got, err := applyUpdate([]byte(`<Root><Key /></Root>`), update)
		require.NoError(t, err)

		assert.Equal(t, `<Root><Key >new</Key></Root>`, string(got))
		_, err = planEdits(bytes.NewReader(got), int64(len(got)), nil)
		assert.NoError(t, err, "updated XML must remain valid")
	})

	t.Run("empty element", func(t *testing.T) {
		t.Parallel()

		update := Update{ID: "test", Path: "/Root/Key", Value: "new"}

		got, err := applyUpdate([]byte(`<Root><Key></Key></Root>`), update)
		require.NoError(t, err)

		assert.Equal(t, `<Root><Key>new</Key></Root>`, string(got))
		_, err = planEdits(bytes.NewReader(got), int64(len(got)), nil)
		assert.NoError(t, err, "updated XML must remain valid")
	})

	t.Run("empty replacement", func(t *testing.T) {
		t.Parallel()

		update := Update{ID: "test", Path: "/Root/Key", Value: ""}

		got, err := applyUpdate([]byte(`<Root><Key>old</Key></Root>`), update)
		require.NoError(t, err)

		assert.Equal(t, `<Root><Key></Key></Root>`, string(got))
		_, err = planEdits(bytes.NewReader(got), int64(len(got)), nil)
		assert.NoError(t, err, "updated XML must remain valid")
	})

	t.Run("escaped text", func(t *testing.T) {
		t.Parallel()

		update := Update{ID: "test", Path: "/Root/Key", Value: "a<&"}

		got, err := applyUpdate([]byte(`<Root><Key>old</Key></Root>`), update)
		require.NoError(t, err)

		assert.Equal(t, `<Root><Key>a&lt;&amp;</Key></Root>`, string(got))
		_, err = planEdits(bytes.NewReader(got), int64(len(got)), nil)
		assert.NoError(t, err, "updated XML must remain valid")
	})

	t.Run("comment and CDATA", func(t *testing.T) {
		t.Parallel()

		update := Update{ID: "test", Path: "/Root/Key", Value: "new"}

		got, err := applyUpdate([]byte(`<?xml version="1.0"?><Root><!--keep--><Key><![CDATA[old]]><!--inner--> text</Key></Root>`), update)
		require.NoError(t, err)

		assert.Equal(t, `<?xml version="1.0"?><Root><!--keep--><Key>new<!--inner--></Key></Root>`, string(got))
		_, err = planEdits(bytes.NewReader(got), int64(len(got)), nil)
		assert.NoError(t, err, "updated XML must remain valid")
	})

	t.Run("semantic no op", func(t *testing.T) {
		t.Parallel()

		update := Update{ID: "test", Path: "/Root/Key", Value: "a&b"}

		got, err := applyUpdate([]byte(`<Root><Key><![CDATA[a&b]]></Key></Root>`), update)
		require.NoError(t, err)

		assert.Equal(t, `<Root><Key><![CDATA[a&b]]></Key></Root>`, string(got))
		_, err = planEdits(bytes.NewReader(got), int64(len(got)), nil)
		assert.NoError(t, err, "updated XML must remain valid")
	})

	t.Run("positional", func(t *testing.T) {
		t.Parallel()

		update := Update{ID: "test", Path: "/Root/Key[2]", Value: "c"}

		got, err := applyUpdate([]byte(`<Root><Key>a</Key><Key>b</Key></Root>`), update)
		require.NoError(t, err)

		assert.Equal(t, `<Root><Key>a</Key><Key>c</Key></Root>`, string(got))
		_, err = planEdits(bytes.NewReader(got), int64(len(got)), nil)
		assert.NoError(t, err, "updated XML must remain valid")
	})

	t.Run("attribute predicate", func(t *testing.T) {
		t.Parallel()

		update := Update{ID: "test", Path: `/Root/Key[@name='a/b']`, Value: "c"}

		got, err := applyUpdate([]byte(`<Root><Key name="other">a</Key><Key name="a/b">b</Key></Root>`), update)
		require.NoError(t, err)

		assert.Equal(t, `<Root><Key name="other">a</Key><Key name="a/b">c</Key></Root>`, string(got))
		_, err = planEdits(bytes.NewReader(got), int64(len(got)), nil)
		assert.NoError(t, err, "updated XML must remain valid")
	})

	t.Run("qualified names", func(t *testing.T) {
		t.Parallel()

		update := Update{ID: "test", Path: "/p:Root/p:Key/@q:name", Value: "new"}

		got, err := applyUpdate([]byte(`<p:Root xmlns:p="urn:r" xmlns:q="urn:a"><p:Key q:name="old">v</p:Key></p:Root>`), update)
		require.NoError(t, err)

		assert.Equal(t, `<p:Root xmlns:p="urn:r" xmlns:q="urn:a"><p:Key q:name="new">v</p:Key></p:Root>`, string(got))
		_, err = planEdits(bytes.NewReader(got), int64(len(got)), nil)
		assert.NoError(t, err, "updated XML must remain valid")
	})

	t.Run("default namespace", func(t *testing.T) {
		t.Parallel()

		update := Update{ID: "test", Path: "/Root/Key", Value: "new"}

		got, err := applyUpdate([]byte(`<Root xmlns="urn:r"><Key>old</Key></Root>`), update)
		require.NoError(t, err)

		assert.Equal(t, `<Root xmlns="urn:r"><Key>new</Key></Root>`, string(got))
		_, err = planEdits(bytes.NewReader(got), int64(len(got)), nil)
		assert.NoError(t, err, "updated XML must remain valid")
	})

	t.Run("ambiguous", func(t *testing.T) {
		t.Parallel()

		update := Update{ID: "test", Path: "/Root/Key", Value: ""}

		_, err := applyUpdate([]byte(`<Root><Key>a</Key><Key>b</Key></Root>`), update)
		assert.Error(t, err)
	})

	t.Run("missing", func(t *testing.T) {
		t.Parallel()

		update := Update{ID: "test", Path: "/Root/Key", Value: ""}

		_, err := applyUpdate([]byte(`<Root/>`), update)
		assert.Error(t, err)
	})

	t.Run("missing attribute", func(t *testing.T) {
		t.Parallel()

		update := Update{ID: "test", Path: "/Root/@host", Value: ""}

		_, err := applyUpdate([]byte(`<Root/>`), update)
		assert.Error(t, err)
	})

	t.Run("non leaf", func(t *testing.T) {
		t.Parallel()

		update := Update{ID: "test", Path: "/Root", Value: ""}

		_, err := applyUpdate([]byte(`<Root><Key>v</Key></Root>`), update)
		assert.Error(t, err)
	})

	t.Run("invalid text", func(t *testing.T) {
		t.Parallel()

		update := Update{ID: "test", Path: "/Root", Value: "bad\x00text"}

		_, err := applyUpdate([]byte(`<Root/>`), update)
		assert.Error(t, err)
	})

	t.Run("multiple roots", func(t *testing.T) {
		t.Parallel()

		update := Update{ID: "test", Path: "/Root", Value: ""}

		_, err := applyUpdate([]byte(`<Root/><Root/>`), update)
		assert.Error(t, err)
	})

	t.Run("malformed", func(t *testing.T) {
		t.Parallel()

		update := Update{ID: "test", Path: "/Root/Key", Value: ""}

		_, err := applyUpdate([]byte(`<Root><Key></Root>`), update)
		assert.Error(t, err)
	})

	t.Run("evaluates selectors against original document", func(t *testing.T) {
		t.Parallel()

		source := []byte(`<Root><Key name="old">old</Key></Root>`)
		reader := bytes.NewReader(source)
		updates := []Update{
			{ID: "rename", Path: "/Root/Key/@name", Value: "new"},
			{ID: "text", Path: "/Root/Key[@name='old']", Value: "new"},
		}

		edits, err := planEdits(reader, int64(len(source)), updates)
		require.NoError(t, err)
		var result bytes.Buffer
		require.NoError(t, copyEdits(reader, int64(len(source)), &result, edits))

		assert.Equal(t, `<Root><Key name="new">new</Key></Root>`, result.String())
	})
}

func TestRejectUnsupportedSelectors(t *testing.T) {
	t.Parallel()

	t.Run("empty selector", func(t *testing.T) {
		t.Parallel()

		path := ""
		update := Update{ID: "test", Path: path, Value: "new"}

		_, err := applyUpdate([]byte(`<Root><Key>old</Key></Root>`), update)
		assert.Error(t, err)
	})

	t.Run("relative selector", func(t *testing.T) {
		t.Parallel()

		path := "Root/Key"
		update := Update{ID: "test", Path: path, Value: "new"}

		_, err := applyUpdate([]byte(`<Root><Key>old</Key></Root>`), update)
		assert.Error(t, err)
	})

	t.Run("root slash only", func(t *testing.T) {
		t.Parallel()

		path := "/"
		update := Update{ID: "test", Path: path, Value: "new"}

		_, err := applyUpdate([]byte(`<Root><Key>old</Key></Root>`), update)
		assert.Error(t, err)
	})

	t.Run("descendant axis", func(t *testing.T) {
		t.Parallel()

		path := "//Key"
		update := Update{ID: "test", Path: path, Value: "new"}

		_, err := applyUpdate([]byte(`<Root><Key>old</Key></Root>`), update)
		assert.Error(t, err)
	})

	t.Run("trailing slash", func(t *testing.T) {
		t.Parallel()

		path := "/Root/"
		update := Update{ID: "test", Path: path, Value: "new"}

		_, err := applyUpdate([]byte(`<Root><Key>old</Key></Root>`), update)
		assert.Error(t, err)
	})

	t.Run("zero index", func(t *testing.T) {
		t.Parallel()

		path := "/Root/Key[0]"
		update := Update{ID: "test", Path: path, Value: "new"}

		_, err := applyUpdate([]byte(`<Root><Key>old</Key></Root>`), update)
		assert.Error(t, err)
	})

	t.Run("negative index", func(t *testing.T) {
		t.Parallel()

		path := "/Root/Key[-1]"
		update := Update{ID: "test", Path: path, Value: "new"}

		_, err := applyUpdate([]byte(`<Root><Key>old</Key></Root>`), update)
		assert.Error(t, err)
	})

	t.Run("unclosed predicate", func(t *testing.T) {
		t.Parallel()

		path := "/Root/Key["
		update := Update{ID: "test", Path: path, Value: "new"}

		_, err := applyUpdate([]byte(`<Root><Key>old</Key></Root>`), update)
		assert.Error(t, err)
	})

	t.Run("unopened predicate", func(t *testing.T) {
		t.Parallel()

		path := "/Root/Key]"
		update := Update{ID: "test", Path: path, Value: "new"}

		_, err := applyUpdate([]byte(`<Root><Key>old</Key></Root>`), update)
		assert.Error(t, err)
	})

	t.Run("unquoted predicate value", func(t *testing.T) {
		t.Parallel()

		path := "/Root/Key[@name=bad]"
		update := Update{ID: "test", Path: path, Value: "new"}

		_, err := applyUpdate([]byte(`<Root><Key>old</Key></Root>`), update)
		assert.Error(t, err)
	})

	t.Run("unclosed quoted value", func(t *testing.T) {
		t.Parallel()

		path := "/Root/Key[@name='bad]"
		update := Update{ID: "test", Path: path, Value: "new"}

		_, err := applyUpdate([]byte(`<Root><Key>old</Key></Root>`), update)
		assert.Error(t, err)
	})

	t.Run("multiple predicates", func(t *testing.T) {
		t.Parallel()

		path := "/Root/Key[1][2]"
		update := Update{ID: "test", Path: path, Value: "new"}

		_, err := applyUpdate([]byte(`<Root><Key>old</Key></Root>`), update)
		assert.Error(t, err)
	})

	t.Run("nested predicates", func(t *testing.T) {
		t.Parallel()

		path := "/Root/Key[[1]]"
		update := Update{ID: "test", Path: path, Value: "new"}

		_, err := applyUpdate([]byte(`<Root><Key>old</Key></Root>`), update)
		assert.Error(t, err)
	})

	t.Run("root attribute", func(t *testing.T) {
		t.Parallel()

		path := "/@host"
		update := Update{ID: "test", Path: path, Value: "new"}

		_, err := applyUpdate([]byte(`<Root><Key>old</Key></Root>`), update)
		assert.Error(t, err)
	})

	t.Run("empty attribute", func(t *testing.T) {
		t.Parallel()

		path := "/Root/@"
		update := Update{ID: "test", Path: path, Value: "new"}

		_, err := applyUpdate([]byte(`<Root><Key>old</Key></Root>`), update)
		assert.Error(t, err)
	})

	t.Run("selector union", func(t *testing.T) {
		t.Parallel()

		path := "/Root/Key|/Root/Other"
		update := Update{ID: "test", Path: path, Value: "new"}

		_, err := applyUpdate([]byte(`<Root><Key>old</Key></Root>`), update)
		assert.Error(t, err)
	})
}

func FuzzDocumentUpdates(f *testing.F) {
	f.Add(`<Root/>`, "replacement<&")
	f.Add(`<Root key='value'/>`, "replacement<&")
	f.Add(`<Root>old<!--keep--><![CDATA[text]]></Root>`, "replacement<&")
	f.Add(`<p:Root xmlns:p="urn:test" p:key="value"/>`, "replacement<&")
	f.Add(`<Root><Child/></Root>`, "replacement<&")
	f.Add(`<Root`, "replacement<&")

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
