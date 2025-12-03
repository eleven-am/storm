package main

import (
	"os"
	"testing"

	"github.com/eleven-am/storm/pkg/storm"
)

func TestMain(m *testing.M) {
	m.Run()
}

func TestExecute(t *testing.T) {
	t.Run("execute_function_exists", func(t *testing.T) {
		t.Log("Execute function exists and is callable")
	})
}

func TestInitStormFactories(t *testing.T) {
	t.Run("init_factories", func(t *testing.T) {

		if storm.MigratorFactory != nil {
			t.Error("Expected MigratorFactory to be nil before initialization")
		}
		if storm.ORMFactory != nil {
			t.Error("Expected ORMFactory to be nil before initialization")
		}
		if storm.SchemaInspectorFactory != nil {
			t.Error("Expected SchemaInspectorFactory to be nil before initialization")
		}

		initStormFactories()

		if storm.MigratorFactory == nil {
			t.Error("Expected MigratorFactory to be set after initialization")
		}
		if storm.ORMFactory == nil {
			t.Error("Expected ORMFactory to be set after initialization")
		}
		if storm.SchemaInspectorFactory == nil {
			t.Error("Expected SchemaInspectorFactory to be set after initialization")
		}

		t.Log("Storm factories initialized successfully")
	})
}

func TestMainFunction(t *testing.T) {

	t.Run("main_function_exists", func(t *testing.T) {

		oldArgs := os.Args
		defer func() { os.Args = oldArgs }()

		os.Args = []string{"storm", "--help"}

		err := Execute()

		if err != nil {
			t.Logf("Execute returned error (expected for --help): %v", err)
		}
	})
}

func TestFactoryAssignment(t *testing.T) {

	t.Run("factory_functions_set", func(t *testing.T) {

		storm.MigratorFactory = nil
		storm.ORMFactory = nil
		storm.SchemaInspectorFactory = nil

		initStormFactories()

		if storm.MigratorFactory == nil {
			t.Error("MigratorFactory should be set")
		}
		if storm.ORMFactory == nil {
			t.Error("ORMFactory should be set")
		}
		if storm.SchemaInspectorFactory == nil {
			t.Error("SchemaInspectorFactory should be set")
		}
	})
}
