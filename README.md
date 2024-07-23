# yangSchemaGen
This tool will generate a schema file from a set of yang files.

## Usage
```
↪ ./yangSchemaGen -h
Usage of ./yangSchemaGen:
  -outfile string
        output json file that contains the schema from the input yang (default "schema.json")
  -skipmodules string
        comma separated set of modules to skip, e.g. 'ietf-interfaces'
```

Specifying the outfile will put the schema in the desired location/filename.

Using -skipmodules will avoid generating automatically pulled in modules (ietf-interfaces being the main culprit here).

I only test this with OC models.

Only config schemas are generated.

All subsequent arguments are treated as input filenames for yang parsing.

Union types are sketchy here, I just treat them as strings for now.


