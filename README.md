# f'n Go!

This Golang package provides support for functional-style processing pipelines using [generics](https://go.dev/blog/intro-generics). Each stage of a pipeline runs in a goroutine and is connected by channels. The failure of any stage aborts the chain and returns an error. An abort may also be triggered manually using a [Context](https://pkg.go.dev/context).

## Usage

Every pipeline begins with a `Source` producing a sequence of typed values and ends with a `Sink` consuming a sequence of values. In between these can be any combination of processing functions, of which the following are presently supported:

- `Filter` -- Removes values according to a rule
- `Flatten` -- Collapses a slice-of-slices into a simple slice (e.g. `[[1,2,3],[4,5,6]]` becomes `[1,2,3,4,5,6]`)
- `Map` -- Converts values into something new according to a rule

A parallelized version of Map also exists but does not guarantee the sequence of values is maintained.

## Example

In this demonstration a series of names (`SliceSource`) are reduced to only those containing the letter A (`Filter`). The remaining values are converted to their corresponding lengths (`Map`), and the resulting sequence of numbers is printed to standard-out (`Sink`).

```
func main() {
    names := functional.SliceSource(context.Background(),
        []string{"alice", "bob", "charlie", "david", "erin"})

    namesWithA := functional.Filter(names, onlyWithA)
    lengths := functional.Map(namesWithA, nameLength)

    err := functional.Sink(lengths, printLength)
    if err != nil {
        log.Fatal(err)
    }
}

func nameLength(ctx context.Context, name string) (int, error) {
    return len(name), nil
}

func onlyWithA(ctx context.Context, name string) (bool, error) {
    return strings.HasRune(name, 'a'), nil
}

func printLength(ctx context.Context, length int) error {
    fmt.Println(length)
    return nil
}

func someNames(ctx context.Context, output chan<- string) error {
    for name := range  {
        output <- name
    }
    return nil
}
```

The final output is as follows:

```
5
7
5
```
