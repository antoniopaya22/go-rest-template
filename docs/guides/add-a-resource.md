# Add a resource

A resource is a feature package, one migration per database driver, its part of the
OpenAPI document and two lines of wiring. The generator writes all of it by cloning the
`tasks` example under a new name.

## Generate it

```sh
make resource name=project
```

```text
Created the projects resource:
  internal/projects/api_test.go
  internal/projects/handler.go
  internal/projects/repository.go
  internal/projects/service.go
  internal/projects/project.go
  migrations/postgres/00003_projects.sql
  migrations/sqlite/00003_projects.sql

Updated api/openapi.yaml, internal/app/app.go and internal/app/routes.go.
```

The project builds and its tests pass straight away:

```sh
make check
```

You now have `GET`, `POST` on `/api/v1/projects` and `GET`, `PATCH`, `DELETE` on
`/api/v1/projects/{projectID}`, with the same fields and rules as tasks.

### Names

`name` is the singular, as **one lowercase word**. The plural is derived by adding `s`,
with `y` becoming `ies` after a consonant and `es` added after `s`, `x`, `ch` and `sh`.
Override it when that is wrong:

```sh
make resource name=person plural=people
```

Names with several words, such as `blog_post`, are not supported. Generate `post` and
rename, or copy the package by hand.

## What it writes

| Where | What |
|---|---|
| `internal/<plural>/` | A copy of `internal/tasks` with every `task`, `tasks`, `Task` and `Tasks` renamed. |
| `migrations/<driver>/NNNNN_<plural>.sql` | A copy of the tasks migration with the next free number, for each driver present. |
| `api/openapi.yaml` | The tag, the paths, the response and the schemas of tasks, renamed and inserted at the `# scaffold:` anchors. |
| `internal/app/app.go` | The import, the handler field and its construction. |
| `internal/app/routes.go` | The `Routes` call. |

The generator finds its insertion points through comments such as `// scaffold:routes` and
copies the blocks between `# scaffold:tasks:paths:begin` and `:end`. Keep those comments if
you want to keep generating resources.

## Make it yours

The generated resource has the fields of a task: `title`, `description`, `status` and an
owner. Change it in this order.

**1. The migration.** Edit the new files under `migrations/postgres` and
`migrations/sqlite`. Keep `id`, `version`, `created_at` and `updated_at`; the conventions
rely on them. Keep the owner column if the resource belongs to a user.

**2. The model.** In `internal/projects/project.go`, change the struct, the `Filter`, and
the `CreateInput` and `UpdateInput` types. `UpdateInput` uses pointers so that an absent
field means "leave unchanged".

**3. The repository.** Update the `columns` constant, the `scan` function and the
`INSERT` and `UPDATE` statements in `repository.go`. Queries use `?` placeholders for both
databases.

**4. The service.** Replace the validation and the rules in `service.go`.

**5. The contract.** Update the schemas in `api/openapi.yaml` to match the model:
`Project`, `ProjectCreate`, `ProjectUpdate`, `ProjectPage` and `ProjectStatus`.

**6. The tests.** Adjust `api_test.go`.

Then run `make check`.

## What keeps you honest

Two things fail when the pieces drift apart.

**The parity test.** `TestRoutesMatchContract` in `internal/app/app_test.go` lists the
routes of the router and the operations of the OpenAPI document and requires them to be
equal. A route without documentation, or documentation without a route, fails the test.

**Response validation.** The end-to-end tests run with `OPENAPI_VALIDATE_RESPONSES=true`.
A handler that returns a field the schema does not allow, or omits a required one, gets a
500 in the test instead of passing unnoticed.

## Resources that do not belong to a user

The example scopes everything to an owner. For a resource without one:

- Remove `user_id` from the migration, the model, the repository and the schemas.
- In `handler.go`, remove the calls to `Authorize` in `create` and `load`, and the
  `user_id` filter in `list`.
- Decide who may write. To restrict writes to administrators, put those routes in a group
  with `r.Use(auth.RequireRole(auth.RoleAdmin))`.

## Without the generator

If you removed the example, write the package by hand following the same four files, add
the migration and the OpenAPI paths, then wire it up in `internal/app/app.go`:

```go
a.projects = projects.NewHandler(projects.NewService(projects.NewRepository(a.db)))
```

and in `internal/app/routes.go`, inside the `/api/v1` route:

```go
a.projects.Routes(r, a.authn)
```
