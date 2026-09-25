package monitor

import (
	"testing"
)

func TestNewStatusesStore(t *testing.T) {
	store := NewStatusesStore()

	if store == nil {
		t.Fatal("expected store, got nil")
	}

	if store.statuses == nil {
		t.Fatal("expected initialized statuses map")
	}

	expected := map[string]CollectorStatus{
		"cpu": {
			Required: true,
			State:    CollectorUnknown,
		},
		"memory": {
			Required: true,
			State:    CollectorUnknown,
		},
		"network": {
			Required: true,
			State:    CollectorUnknown,
		},
		"temperature": {
			Required: false,
			State:    CollectorUnknown,
		},
	}

	if len(store.statuses) != len(expected) {
		t.Fatalf(
			"statuses length = %d, want %d",
			len(store.statuses),
			len(expected),
		)
	}

	for collector, expectedStatus := range expected {
		got, ok := store.statuses[collector]
		if !ok {
			t.Fatalf("missing collector %q", collector)
		}

		if got != expectedStatus {
			t.Fatalf(
				"statuses[%q] = %+v, want %+v",
				collector,
				got,
				expectedStatus,
			)
		}
	}
}

func TestCollectorStatusStore_Set(t *testing.T) {
	t.Run("guarda un estado correctamente", func(t *testing.T) {
		store := NewStatusesStore()

		expected := CollectorStatus{
			Required: true,
			State:    CollectorAvailable,
		}

		err := store.Set("cpu", expected)
		if err != nil {
			t.Fatalf("Set() unexpected error: %v", err)
		}

		got, err := store.Get("cpu")
		if err != nil {
			t.Fatalf("Get() unexpected error: %v", err)
		}

		if got != expected {
			t.Fatalf("Get() = %+v, want %+v", got, expected)
		}
	})

	t.Run("actualiza un estado existente", func(t *testing.T) {
		store := NewStatusesStore()

		expected := CollectorStatus{
			Required: true,
			State:    CollectorAvailable,
		}

		err := store.Set("cpu", expected)
		if err != nil {
			t.Fatalf("Set() unexpected error: %v", err)
		}

		got, err := store.Get("cpu")
		if err != nil {
			t.Fatalf("Get() unexpected error: %v", err)
		}

		if got != expected {
			t.Fatalf("Get() = %+v, want %+v", got, expected)
		}
	})

	t.Run("rechaza un collector vacío", func(t *testing.T) {
		store := NewStatusesStore()

		err := store.Set("", CollectorStatus{})
		if err == nil {
			t.Fatal("Set() expected error for empty collector name")
		}

		if err.Error() != "collector name cannot be empty" {
			t.Fatalf(
				"Set() error = %q, want %q",
				err.Error(),
				"collector name cannot be empty",
			)
		}
	})

	t.Run("rechaza un collector inválido", func(t *testing.T) {
		store := NewStatusesStore()

		err := store.Set("disk", CollectorStatus{})
		if err == nil {
			t.Fatal("Set() expected error for invalid collector name")
		}

		if err.Error() != "invalid collector name disk" {
			t.Fatalf(
				"Set() error = %q, want %q",
				err.Error(),
				"invalid collector name disk",
			)
		}
	})
}

func TestCollectorStatusStore_Get(t *testing.T) {
	t.Run("obtiene un estado correctamente", func(t *testing.T) {
		store := NewStatusesStore()

		expected := CollectorStatus{
			Required: true,
			State:    CollectorAvailable,
		}

		if err := store.Set("cpu", expected); err != nil {
			t.Fatalf("Set() unexpected error: %v", err)
		}

		got, err := store.Get("cpu")
		if err != nil {
			t.Fatalf("Get() unexpected error: %v", err)
		}

		if got != expected {
			t.Fatalf("Get() = %+v, want %+v", got, expected)
		}
	})

	t.Run("obtiene los estados iniciales", func(t *testing.T) {
		store := NewStatusesStore()

		expected := map[string]CollectorStatus{
			"cpu": {
				Required: true,
				State:    CollectorUnknown,
			},
			"memory": {
				Required: true,
				State:    CollectorUnknown,
			},
			"network": {
				Required: true,
				State:    CollectorUnknown,
			},
			"temperature": {
				Required: false,
				State:    CollectorUnknown,
			},
		}

		for collector, expectedStatus := range expected {
			got, err := store.Get(collector)
			if err != nil {
				t.Fatalf("Get(%q) unexpected error: %v", collector, err)
			}

			if got != expectedStatus {
				t.Fatalf(
					"Get(%q) = %+v, want %+v",
					collector,
					got,
					expectedStatus,
				)
			}
		}
	})

	t.Run("rechaza un collector vacío", func(t *testing.T) {
		store := NewStatusesStore()

		_, err := store.Get("")
		if err == nil {
			t.Fatal("Get() expected error for empty collector name")
		}

		if err.Error() != "collector name cannot be empty" {
			t.Fatalf(
				"Get() error = %q, want %q",
				err.Error(),
				"collector name cannot be empty",
			)
		}
	})

	t.Run("rechaza un collector inválido", func(t *testing.T) {
		store := NewStatusesStore()

		_, err := store.Get("disk")
		if err == nil {
			t.Fatal("Get() expected error for invalid collector name")
		}

		if err.Error() != "invalid collector name disk" {
			t.Fatalf(
				"Get() error = %q, want %q",
				err.Error(),
				"invalid collector name disk",
			)
		}
	})
}

func TestCollectorStatusStore_SetState(t *testing.T) {
	t.Run("cambia el estado sin cambiar Required", func(t *testing.T) {
		store := NewStatusesStore()

		initialStatus := CollectorStatus{
			Required: true,
			State:    CollectorUnknown,
		}

		if err := store.Set("cpu", initialStatus); err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		if err := store.SetState("cpu", CollectorAvailable); err != nil {
			t.Fatalf("SetState() error = %v", err)
		}

		status, err := store.Get("cpu")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}

		if status.State != CollectorAvailable {
			t.Errorf(
				"State = %q, want %q",
				status.State,
				CollectorAvailable,
			)
		}

		if status.Required != initialStatus.Required {
			t.Errorf(
				"Required = %t, want %t",
				status.Required,
				initialStatus.Required,
			)
		}
	})

	t.Run("rechaza un collector vacío", func(t *testing.T) {
		store := NewStatusesStore()

		err := store.SetState("", CollectorAvailable)
		if err == nil {
			t.Fatal("SetState() expected error for empty collector name")
		}

		if err.Error() != "collector name cannot be empty" {
			t.Fatalf(
				"SetState() error = %q, want %q",
				err.Error(),
				"collector name cannot be empty",
			)
		}
	})

	t.Run("rechaza un collector inválido", func(t *testing.T) {
		store := NewStatusesStore()

		err := store.SetState("disk", CollectorAvailable)
		if err == nil {
			t.Fatal("SetState() expected error for invalid collector name")
		}

		if err.Error() != "invalid collector name disk" {
			t.Fatalf(
				"SetState() error = %q, want %q",
				err.Error(),
				"invalid collector name disk",
			)
		}
	})

	t.Run("permite cambiar entre diferentes estados", func(t *testing.T) {
		store := NewStatusesStore()

		states := []CollectorState{
			CollectorAvailable,
			CollectorUnavailable,
			CollectorNotSupported,
			CollectorUnknown,
		}

		for _, expectedState := range states {
			if err := store.SetState("cpu", expectedState); err != nil {
				t.Fatalf(
					"SetState(%q) error = %v",
					expectedState,
					err,
				)
			}

			got, err := store.Get("cpu")
			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}

			if got.State != expectedState {
				t.Fatalf(
					"State = %q, want %q",
					got.State,
					expectedState,
				)
			}

			if !got.Required {
				t.Fatal("Required changed unexpectedly")
			}
		}
	})
}

func TestCollectorStatusStore_GetAll(t *testing.T) {
	t.Run("devuelve todos los estados", func(t *testing.T) {
		store := NewStatusesStore()

		statuses := store.GetAll()

		if len(statuses) != 4 {
			t.Fatalf(
				"GetAll() length = %d, want 4",
				len(statuses),
			)
		}

		expected := map[string]CollectorStatus{
			"cpu": {
				Required: true,
				State:    CollectorUnknown,
			},
			"memory": {
				Required: true,
				State:    CollectorUnknown,
			},
			"network": {
				Required: true,
				State:    CollectorUnknown,
			},
			"temperature": {
				Required: false,
				State:    CollectorUnknown,
			},
		}

		for collector, expectedStatus := range expected {
			got, ok := statuses[collector]
			if !ok {
				t.Fatalf("GetAll() missing collector %q", collector)
			}

			if got != expectedStatus {
				t.Fatalf(
					"GetAll()[%q] = %+v, want %+v",
					collector,
					got,
					expectedStatus,
				)
			}
		}
	})

	t.Run("devuelve un mapa independiente del store", func(t *testing.T) {
		store := NewStatusesStore()

		statuses := store.GetAll()
		delete(statuses, "cpu")

		_, err := store.Get("cpu")
		if err != nil {
			t.Fatalf(
				"Get() returned error after modifying GetAll() result: %v",
				err,
			)
		}
	})

	t.Run("permite modificar el resultado sin afectar el store", func(t *testing.T) {
		store := NewStatusesStore()

		expected, err := store.Get("cpu")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}

		statuses := store.GetAll()
		statuses["cpu"] = CollectorStatus{
			Required: false,
			State:    CollectorUnavailable,
		}

		got, err := store.Get("cpu")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}

		if got != expected {
			t.Fatalf(
				"store was modified through GetAll(): got %+v, want %+v",
				got,
				expected,
			)
		}
	})
}

func TestCollectorStatusStore_Ready(t *testing.T) {
	t.Run("no está listo inicialmente", func(t *testing.T) {
		store := NewStatusesStore()

		if store.Ready() {
			t.Fatal("Ready() = true, want false")
		}
	})

	t.Run("está listo cuando todos los collectors requeridos están disponibles", func(t *testing.T) {
		store := NewStatusesStore()

		requiredCollectors := []string{
			"cpu",
			"memory",
			"network",
		}

		for _, collector := range requiredCollectors {
			if err := store.SetState(collector, CollectorAvailable); err != nil {
				t.Fatalf(
					"SetState(%q) error = %v",
					collector,
					err,
				)
			}
		}

		if !store.Ready() {
			t.Fatal("Ready() = false, want true")
		}
	})

	t.Run("no está listo si un collector requerido está unavailable", func(t *testing.T) {
		store := NewStatusesStore()

		if err := store.SetState("cpu", CollectorAvailable); err != nil {
			t.Fatalf("SetState(cpu) error = %v", err)
		}

		if err := store.SetState("memory", CollectorUnavailable); err != nil {
			t.Fatalf("SetState(memory) error = %v", err)
		}

		if err := store.SetState("network", CollectorAvailable); err != nil {
			t.Fatalf("SetState(network) error = %v", err)
		}

		if store.Ready() {
			t.Fatal("Ready() = true, want false")
		}
	})

	t.Run("no está listo si un collector requerido está not supported", func(t *testing.T) {
		store := NewStatusesStore()

		if err := store.SetState("cpu", CollectorNotSupported); err != nil {
			t.Fatalf("SetState(cpu) error = %v", err)
		}

		if err := store.SetState("memory", CollectorAvailable); err != nil {
			t.Fatalf("SetState(memory) error = %v", err)
		}

		if err := store.SetState("network", CollectorAvailable); err != nil {
			t.Fatalf("SetState(network) error = %v", err)
		}

		if store.Ready() {
			t.Fatal("Ready() = true, want false")
		}
	})

	t.Run("no está listo si un collector requerido está unknown", func(t *testing.T) {
		store := NewStatusesStore()

		if err := store.SetState("cpu", CollectorAvailable); err != nil {
			t.Fatalf("SetState(cpu) error = %v", err)
		}

		if err := store.SetState("memory", CollectorUnknown); err != nil {
			t.Fatalf("SetState(memory) error = %v", err)
		}

		if err := store.SetState("network", CollectorAvailable); err != nil {
			t.Fatalf("SetState(network) error = %v", err)
		}

		if store.Ready() {
			t.Fatal("Ready() = true, want false")
		}
	})

	t.Run("está listo aunque un collector opcional no esté soportado", func(t *testing.T) {
		store := NewStatusesStore()

		requiredCollectors := []string{
			"cpu",
			"memory",
			"network",
		}

		for _, collector := range requiredCollectors {
			if err := store.SetState(collector, CollectorAvailable); err != nil {
				t.Fatalf(
					"SetState(%q) error = %v",
					collector,
					err,
				)
			}
		}

		if err := store.SetState("temperature", CollectorNotSupported); err != nil {
			t.Fatalf("SetState(temperature) error = %v", err)
		}

		if !store.Ready() {
			t.Fatal("Ready() = false, want true")
		}
	})

	t.Run("está listo aunque un collector opcional esté unavailable", func(t *testing.T) {
		store := NewStatusesStore()

		requiredCollectors := []string{
			"cpu",
			"memory",
			"network",
		}

		for _, collector := range requiredCollectors {
			if err := store.SetState(collector, CollectorAvailable); err != nil {
				t.Fatalf(
					"SetState(%q) error = %v",
					collector,
					err,
				)
			}
		}

		if err := store.SetState("temperature", CollectorUnavailable); err != nil {
			t.Fatalf("SetState(temperature) error = %v", err)
		}

		if !store.Ready() {
			t.Fatal("Ready() = false, want true")
		}
	})
}
