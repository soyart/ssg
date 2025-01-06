# ssg-go

ssg-go is a drop-in replacement and library for implementing ssg.

## Extending and consuming ssg-go

### ssg-go walk

Given `src` and `dst` paths, ssg-go walks `src` and performs the
following operations for each source file:

- If path is ignored

  ssg-go continues to the next input.

- If path is unignored directory

  ssg-go collects templates from `_header.html` and `_footer.html`

- If path is a file

  ssg-go reads the data and send it to all of the `Pipeline`s.

  The output from the last `Pipeline` is used as input to core:

  ```
  raw_data -> [pipelines] -> core
  ```

  Pipelines can use these 2 well known errors control ssg-go walk control flow:

  - `ErrBreakPipelines`

    ssg-go stops going through pipelines and immedietly advances to core

  - `ErrSkipCore`

    Like with `ErrBreakPipelines`, but ssg-go also skips core.

### ssg-go core

For an input file, ssg-go performs these actions:

- If path is a file

  ssg-go calls `Hook` on the file to modify the data.
  We can use minifiers here.

- If path has non-`.md` extension

  ssg-go will simply mirrors the file to `$dst`:

  ```
  raw data post-pipeline -> hook -> output
  ```

- If path has `.md` extension

  ssg-go assembles and adds the HTML output to the outputs.
  After the assembly, `HookGenerate` is called on the data.

  ```
  raw data post-pipeline -> hook -> generate/assemble HTML -> hookGenerate -> output
  ```

### Options

Go programmers can extend ssg-go via its [`Option` type](./options.go).

[soyweb](./soyweb/) also extends ssg via `Option`,
and provides extra functionality such as index generator and minifiers.

Extending via options can be done in 2 rough categories:

- Hooks

  Hooks modify file content in-place in memory (without renaming input files).
  In short, it maps the input bytes 1-1 to new input bytes.

  A good usecase for hooks would be minifiers or content filter.

- Pipelines

  Pipelines have full control of the walk, and can arbitarily generate new
  ssg-go outputs and change input properties such as filenames and modes.

  In soyweb, pipelines are used to implement the [index generator](./index.go).

#### `Hook` option

`Hook` is a Go function used to modify data after it is read,
preserving the filename. `Hook` is only called on the raw inputs
but not on the generated HTMLs.

It is enabled with `WithHook(hook)`

#### `HookGenerate` option

`HookGenerate` is a Go function called on every generated HTML.
For example, soyweb uses this option to implement output minifier.

It is enabled with `WithHookGenerate(hook)`

#### `Pipeline` option

`Pipeline` is a Go function called on a file during directory walk.
To reduce complexity, ignored files and ssg headers/footers are not sent
to `Pipeline`. This preserves the core functionality of the original ssg.

Pipelines can be chained together with `WithPipelines(p1, p2, p3)`

`WithPipelines` accepts 2 types of spread parameters:

- `Pipeline`

  An alias to `func(path string, data []byte, d fs.DirEntry) (string, []byte, fs.DirEntry, error)`
  We can call these pipelines *pure* pipelines.

  An example would be this `Pipeline`, that reads atime from filesystems
  and prepends the date string to Markdowns:

  ```go
  func pipelineUpdatedAt(path string, data []byte, d fs.DirEntry) (string, []byte, fs.DirEntry, error) {
  	if d.IsDir() {
  		return path, data, d, nil
  	}
  	if filepath.Ext(path) != ".md" {
  		return path, data, d, nil
  	}
  	info, err := d.Info()
  	if err != nil {
  		return "", nil, nil, err
  	}
  	buf := bytes.NewBufferString(
  		fmt.Sprintf("%s", info.ModTime().Format(time.DateOnly)),
  	)

  	buf.ReadFrom(bytes.NewBuffer(data))
  	return path, buf.Bytes(), d, nil
  }
  ```

- `func(s *Ssg) Pipeline`

  Some pipelines are not pure and might need something from `Ssg`, so
  we allow these pipeline constructors as arguments to `WithPipelines`.

  An example for this type of pipelines would be the [index generator](./index.go),
  which needs to know which files are ignored in addition to `$src` and `$dst`.

### Streaming and caching builds

To minimize runtime memory usage, ssg-go builds and writes concurrently.
There're 2 main ssg threads: one is for building the outputs,
and the other is the write thread.

The build thread *sequentially* reads, builds and sends outputs
to the write thread via a buffered Go channel.

Bufffering allows the builder thread to continue to build and send outputs
to the writer until the buffer is full.

This helps reduce back pressure, and keeps memory usage low.
The buffer size is, by default, 2x of the number of writers.

This means that, at any point in time during a generation of any number of files
with 20 writers, ssg-go will at most only hold 40 output files
in memory (in the buffered channel).

If you are importing ssg-go to your code and you don't want this
streaming behavior, you can use the exposed function `Build`, `WriteOut`,
and `GenerateMetadata`:

```go
files, dist, err := ssg.Build(src, dst, title, url)
if err != nil {
  panic(err)
}

err = ssg.WriteOut(dist)
if err != nil {
  panic(err)
}

err = GenerateMetadata(src, dst, urk, files, dist, time.Time{})
if err != nil {
  panic(err)
}
```

