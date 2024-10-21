package command_test

import (
	"testing"

	"github.com/renderinc/render-cli/pkg/command"
	"github.com/renderinc/render-cli/pkg/pointers"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestParseCommand(t *testing.T) {
	t.Run("parse basic type", func(t *testing.T) {
		type testStruct struct {
			Foo string `cli:"foo"`
		}
		var v testStruct
		cmd := &cobra.Command{}
		cmd.Flags().String("foo", "", "")
		require.NoError(t, cmd.ParseFlags([]string{"--foo", "bar"}))

		err := command.ParseCommand(cmd, []string{}, &v)
		require.NoError(t, err)

		require.Equal(t, "bar", v.Foo)
	})

	t.Run("parse pointer", func(t *testing.T) {
		type testStruct struct {
			Foo *string `cli:"foo"`
		}
		var v testStruct
		cmd := &cobra.Command{}
		cmd.Flags().String("foo", "", "")
		require.NoError(t, cmd.ParseFlags([]string{"--foo", "bar"}))

		err := command.ParseCommand(cmd, []string{}, &v)
		require.NoError(t, err)

		require.Equal(t, "bar", *v.Foo)
	})

	t.Run("parse slice", func(t *testing.T) {
		type testStruct struct {
			Foo []string `cli:"foo"`
		}
		var v testStruct
		cmd := &cobra.Command{}
		cmd.Flags().StringSlice("foo", []string{}, "")
		require.NoError(t, cmd.ParseFlags([]string{"--foo", "bar,baz"}))

		err := command.ParseCommand(cmd, []string{}, &v)
		require.NoError(t, err)

		require.Equal(t, []string{"bar", "baz"}, v.Foo)
	})

	t.Run("arg parsing", func(t *testing.T) {
		t.Run("simple arg", func(t *testing.T) {
			type testStruct struct {
				Foo string `cli:"arg:0"`
			}
			var v testStruct
			cmd := &cobra.Command{}

			err := command.ParseCommand(cmd, []string{"bar"}, &v)
			require.NoError(t, err)

			require.Equal(t, "bar", v.Foo)
		})

		t.Run("pointer arg", func(t *testing.T) {
			type testStruct struct {
				Foo *string `cli:"arg:0"`
			}
			var v testStruct
			cmd := &cobra.Command{}

			err := command.ParseCommand(cmd, []string{"bar"}, &v)
			require.NoError(t, err)

			require.Equal(t, "bar", *v.Foo)
		})
	})
}

func TestInputToString(t *testing.T) {
	t.Run("args", func(t *testing.T) {
		type testStruct struct {
			Foo string  `cli:"arg:0"`
			Bar *string `cli:"arg:1"`
		}

		v := testStruct{Foo: "abc", Bar: pointers.From("def")}
		str, err := command.InputToString(&v)
		require.NoError(t, err)
		require.Equal(t, "abc def", str)
	})

	t.Run("flags", func(t *testing.T) {
		type testStruct struct {
			Foo string `cli:"foo"`
			Bar *int   `cli:"bar"`
		}

		v := testStruct{Foo: "abc", Bar: pointers.From(123)}
		str, err := command.InputToString(&v)
		require.NoError(t, err)
		require.Equal(t, "--foo=abc --bar=123", str)
	})

	t.Run("args and flags", func(t *testing.T) {
		type testStruct struct {
			Foo  string  `cli:"foo"`
			Bar  *int    `cli:"bar"`
			Arg0 string  `cli:"arg:0"`
			Arg1 *string `cli:"arg:1"`
		}

		v := testStruct{
			Foo:  "abc",
			Bar:  pointers.From(123),
			Arg0: "def",
			Arg1: pointers.From("ghi"),
		}
		str, err := command.InputToString(&v)
		require.NoError(t, err)
		require.Equal(t, "def ghi --foo=abc --bar=123", str)
	})
}
