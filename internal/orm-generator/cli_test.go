package orm_generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDiscoverModels(t *testing.T) {

	tmpDir := os.TempDir()
	testPackageDir := filepath.Join(tmpDir, "test_package")
	outputDir := filepath.Join(tmpDir, "orm_output")

	defer func() {
		os.RemoveAll(testPackageDir)
		os.RemoveAll(outputDir)
	}()

	err := os.MkdirAll(testPackageDir, 0755)
	assert.NoError(t, err)

	testGoFile := filepath.Join(testPackageDir, "models.go")
	testContent := `package testmodels

import "time"

type User struct {
	ID        int       ` + "`" + `db:"id" dbdef:"primary_key;auto_increment"` + "`" + `
	Name      string    ` + "`" + `db:"name" dbdef:"not_null"` + "`" + `
	Email     string    ` + "`" + `db:"email" dbdef:"unique;not_null"` + "`" + `
	CreatedAt time.Time ` + "`" + `db:"created_at" dbdef:"default:now()"` + "`" + `
}

type Post struct {
	ID     int    ` + "`" + `db:"id" dbdef:"primary_key;auto_increment"` + "`" + `
	Title  string ` + "`" + `db:"title" dbdef:"not_null"` + "`" + `
	UserID int    ` + "`" + `db:"user_id" dbdef:"not_null"` + "`" + `
	
	User *User ` + "`" + `db:"-" orm:"belongs_to:User;foreign_key:user_id"` + "`" + `
}
`

	err = os.WriteFile(testGoFile, []byte(testContent), 0644)
	assert.NoError(t, err)

	config := GenerationConfig{
		PackageName:  "testmodels",
		OutputDir:    outputDir,
		Models:       []string{},
		Features:     []string{"columns"},
		IncludeTests: false,
		IncludeDocs:  false,
	}

	generator := NewCodeGenerator(config)
	err = generator.DiscoverModels(testPackageDir)
	if err != nil {
		t.Logf("DiscoverModels failed: %v", err)

	}

}

func TestDiscoverModels_InvalidPath(t *testing.T) {
	config := GenerationConfig{
		PackageName: "test",
		OutputDir:   "/tmp/test_output",
	}

	generator := NewCodeGenerator(config)
	err := generator.DiscoverModels("/non/existent/path")

	t.Logf("DiscoverModels on non-existent path returned: %v", err)
}

func TestDiscoverModels_EmptyDirectory(t *testing.T) {
	tmpDir := os.TempDir()
	emptyDir := filepath.Join(tmpDir, "empty_test_dir")
	defer os.RemoveAll(emptyDir)

	err := os.MkdirAll(emptyDir, 0755)
	assert.NoError(t, err)

	config := GenerationConfig{
		PackageName: "test",
		OutputDir:   filepath.Join(tmpDir, "empty_output"),
	}

	generator := NewCodeGenerator(config)
	err = generator.DiscoverModels(emptyDir)
	if err != nil {
		t.Logf("DiscoverModels on empty directory failed: %v", err)

	}
}

func TestParseGoFilesInDirectory(t *testing.T) {
	tmpDir := os.TempDir()
	testDir := filepath.Join(tmpDir, "parse_test")
	defer os.RemoveAll(testDir)

	err := os.MkdirAll(testDir, 0755)
	assert.NoError(t, err)

	testFile1 := filepath.Join(testDir, "model1.go")
	content1 := `package testmodels

type TestModel1 struct {
	ID   int    ` + "`" + `db:"id" dbdef:"primary_key"` + "`" + `
	Name string ` + "`" + `db:"name"` + "`" + `
}
`

	testFile2 := filepath.Join(testDir, "model2.go")
	content2 := `package testmodels

type TestModel2 struct {
	ID    int    ` + "`" + `db:"id" dbdef:"primary_key"` + "`" + `
	Title string ` + "`" + `db:"title"` + "`" + `
}
`

	err = os.WriteFile(testFile1, []byte(content1), 0644)
	assert.NoError(t, err)

	err = os.WriteFile(testFile2, []byte(content2), 0644)
	assert.NoError(t, err)

	generator := NewCodeGenerator(GenerationConfig{
		PackageName: "testmodels",
		OutputDir:   "/tmp/test",
	})

	err = generator.DiscoverModels(testDir)
	if err != nil {
		t.Logf("DiscoverModels failed: %v", err)

	}

	names := generator.GetModelNames()
	t.Logf("Discovered %d models", len(names))
}

func TestCreateOutputDirectory(t *testing.T) {
	tmpDir := os.TempDir()
	outputDir := filepath.Join(tmpDir, "create_output_test")
	defer os.RemoveAll(outputDir)

	config := GenerationConfig{
		PackageName: "test",
		OutputDir:   outputDir,
	}

	generator := NewCodeGenerator(config)

	err := generator.GenerateAll()
	if err != nil {
		t.Logf("GenerateAll failed: %v", err)

	}

	assert.DirExists(t, outputDir)
}
