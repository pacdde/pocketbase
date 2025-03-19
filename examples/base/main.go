package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/ghupdate"
	"github.com/pocketbase/pocketbase/plugins/jsvm"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/pocketbase/pocketbase/tools/geoip"
	"github.com/pocketbase/pocketbase/tools/hook"
	"github.com/pocketbase/pocketbase/tools/types"
)

/*
TODO:

- [file handler 1](https://github.com/pocketbase/pocketbase/blob/master/core/field_file.go#L459)
- [file handler 2](https://github.com/pocketbase/pocketbase/blob/master/core/record_model.go#L602)
- [file handler 3](https://github.com/pocketbase/pocketbase/blob/master/core/record_model.go#L988)
*/
func main() {
	app := pocketbase.New()

	// ---------------------------------------------------------------
	// Optional plugin flags:
	// ---------------------------------------------------------------

	var hooksDir string
	app.RootCmd.PersistentFlags().StringVar(
		&hooksDir,
		"hooksDir",
		"",
		"the directory with the JS app hooks",
	)

	var hooksWatch bool
	app.RootCmd.PersistentFlags().BoolVar(
		&hooksWatch,
		"hooksWatch",
		true,
		"auto restart the app on pb_hooks file change; it has no effect on Windows",
	)

	var hooksPool int
	app.RootCmd.PersistentFlags().IntVar(
		&hooksPool,
		"hooksPool",
		15,
		"the total prewarm goja.Runtime instances for the JS app hooks execution",
	)

	var migrationsDir string
	app.RootCmd.PersistentFlags().StringVar(
		&migrationsDir,
		"migrationsDir",
		"",
		"the directory with the user defined migrations",
	)

	var automigrate bool
	app.RootCmd.PersistentFlags().BoolVar(
		&automigrate,
		"automigrate",
		true,
		"enable/disable auto migrations",
	)

	var publicDir string
	app.RootCmd.PersistentFlags().StringVar(
		&publicDir,
		"publicDir",
		defaultPublicDir(),
		"the directory to serve static files",
	)

	var indexFallback bool
	app.RootCmd.PersistentFlags().BoolVar(
		&indexFallback,
		"indexFallback",
		true,
		"fallback the request to index.html on missing static path, e.g. when pretty urls are used with SPA",
	)

	app.RootCmd.ParseFlags(os.Args[1:])

	// ---------------------------------------------------------------
	// Plugins and hooks:
	// ---------------------------------------------------------------

	// load jsvm (pb_hooks and pb_migrations)
	jsvm.MustRegister(app, jsvm.Config{
		MigrationsDir: migrationsDir,
		HooksDir:      hooksDir,
		HooksWatch:    hooksWatch,
		HooksPoolSize: hooksPool,
	})

	// migrate command (with js templates)
	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		TemplateLang: migratecmd.TemplateLangJS,
		Automigrate:  automigrate,
		Dir:          migrationsDir,
	})

	// GitHub selfupdate
	ghupdate.MustRegister(app, app.RootCmd, ghupdate.Config{})

	app.OnBootstrap().BindFunc(func(e *core.BootstrapEvent) error {
		if err := e.Next(); err != nil {
			return err
		}

		geoip.Init("GeoLite2-City.mmdb")

		return nil
	})

	// static route to serves files from the provided public dir
	// (if publicDir exists and the route path is not already defined)
	app.OnServe().Bind(&hook.Handler[*core.ServeEvent]{
		Func: func(e *core.ServeEvent) error {
			if !e.Router.HasRoute(http.MethodGet, "/{path...}") {
				e.Router.GET("/{path...}", apis.Static(os.DirFS(publicDir), indexFallback))
			}

			return e.Next()
		},
		Priority: 999, // execute as latest as possible to allow users to provide their own route
	})

	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		if _, err := app.FindCollectionByNameOrId("__pb_files__"); err != nil {
			collection := core.NewBaseCollection("files", "__pb_files__")
			// set rules
			collection.ViewRule = types.Pointer("")
			collection.CreateRule = types.Pointer("@request.auth.id != ''")
			collection.UpdateRule = types.Pointer(`
				@request.auth.id != '' &&
				user = @request.auth.id &&
				(@request.body.user:isset = false || @request.body.user = @request.auth.id)
			`)

			collection.Fields.Add(&core.FileField{
				Name:      "filename",
				Required:  true,
				MaxSelect: 1,
				MimeTypes: []string{"image/jpeg", "image/png"},
			})
			collection.Fields.Add(&core.TextField{
				Name:     "password",
				Required: false,
				Max:      32,
				Hidden:   true,
			})
			usersCollection, err := app.FindCollectionByNameOrId("users")
			if err != nil {
				return err
			}
			collection.Fields.Add(&core.RelationField{
				Name:          "user",
				Required:      true,
				CascadeDelete: true,
				CollectionId:  usersCollection.Id,
			})
			collection.Fields.Add(&core.AutodateField{
				Name:     "created",
				OnCreate: true,
			})
			collection.Fields.Add(&core.AutodateField{
				Name:     "updated",
				OnCreate: true,
				OnUpdate: true,
			})
			collection.AddIndex("idx_files_filename", true, "filename", "")
			err = app.Save(collection)
			if err != nil {
				return err
			}
		}
		return e.Next()
	})

	app.OnModelAfterCreateSuccess("_logs").BindFunc(func(e *core.ModelEvent) error {
		if log, ok := e.Model.(*core.Log); ok {
			fmt.Println("log created: ", log.Data)
			// log.Data.Set("7777", 77777)
			// app.Save(log) // try to modify logs data, but not working
		}
		return e.Next()
	})

	app.OnTerminate().BindFunc(func(e *core.TerminateEvent) error {
		geoip.Close()
		return e.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

// the default pb_public dir location is relative to the executable
func defaultPublicDir() string {
	if strings.HasPrefix(os.Args[0], os.TempDir()) {
		// most likely ran with go run
		return "./pb_public"
	}

	return filepath.Join(os.Args[0], "../pb_public")
}
