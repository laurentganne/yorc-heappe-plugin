# Yorc HEAppE plugin

The Yorc HEAppE plugin implements a Yorc ([Ystia orchestrator](https://github.com/ystia/yorc/)) plugin as described in [Yorc documentation](https://yorc.readthedocs.io/en/latest/plugins.html), allowing the orchestrator to use the HEappE (([High-End Application Execution](http://heappe.eu)) API to manage jobs executions on HPC infrastructures.

## To build this plugin

You need first to have a working [Go environment](https://golang.org/doc/install).
Then to build, execute the following instructions:

```
mkdir -p $GOPATH/src/laurentganne
cd $GOPATH/src/laurentganne
git clone https://github.com/laurentganne/yorc-heappe-plugin
cd yorc-heappe-plugin
make
```

## Licensing

This plugin is licensed under the [Apache 2.0 License](LICENSE).

