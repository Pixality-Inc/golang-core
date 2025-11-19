package json

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type TestStruct struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func TestJsonEntity(t *testing.T) {
	t.Parallel()

	t.Run("NewEntity creates entity with marshaled data", func(t *testing.T) {
		t.Parallel()

		obj := TestStruct{Name: "test", Value: 42}
		entity, err := NewEntity(obj)

		require.NoError(t, err)
		require.NotNil(t, entity)
		require.Equal(t, obj, entity.Entity)
		require.NotNil(t, entity.Data)

		// Verify data can be unmarshaled back
		var result TestStruct

		err = Unmarshal(entity.Data, &result)
		require.NoError(t, err)
		require.Equal(t, obj, result)
	})

	t.Run("NewEntityFromByteArray creates entity from json bytes", func(t *testing.T) {
		t.Parallel()

		jsonData := []byte(`{"name":"test","value":42}`)
		entity, err := NewEntityFromByteArray[TestStruct](jsonData)

		require.NoError(t, err)
		require.NotNil(t, entity)
		require.Equal(t, "test", entity.Entity.Name)
		require.Equal(t, 42, entity.Entity.Value)
		require.Equal(t, RawMessage(jsonData), entity.Data)
	})

	t.Run("JsonEntity MarshalJSON returns raw data", func(t *testing.T) {
		t.Parallel()

		entity := &Entity[TestStruct]{
			Entity: TestStruct{Name: "test", Value: 42},
			Data:   RawMessage(`{"name":"test","value":42}`),
		}

		marshaled, err := entity.MarshalJSON()
		require.NoError(t, err)
		require.JSONEq(t, `{"name":"test","value":42}`, string(marshaled))
	})

	t.Run("JsonEntity UnmarshalJSON populates entity and data", func(t *testing.T) {
		t.Parallel()

		entity := &Entity[TestStruct]{}
		jsonData := []byte(`{"name":"test","value":42}`)

		err := entity.UnmarshalJSON(jsonData)
		require.NoError(t, err)
		require.Equal(t, "test", entity.Entity.Name)
		require.Equal(t, 42, entity.Entity.Value)
		require.Equal(t, RawMessage(jsonData), entity.Data)
	})
}
