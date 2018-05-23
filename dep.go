package modprobe

import (
	"bufio"
	"os"
	"strings"

	"pault.ag/go/topsort"
)

// load the dependencies, and dump out the topological sort of the
// modules to use.
func loadOrder(name string) ([]string, error) {
	deps, err := loadDependencies()
	if err != nil {
		return nil, err
	}
	return deps.Load(name)
}

// simple container type that stores a mapping from an element to elements
// that it depends on.
type dependencies map[string][]string

// top level loading of the dependency tree. this will start a network
// walk the dep tree, load them into the network, and return a topological
// sort of the modules.
func (d dependencies) Load(name string) ([]string, error) {
	network := topsort.NewNetwork()
	if err := d.load(name, network); err != nil {
		return nil, err
	}

	order, err := network.Sort()
	if err != nil {
		return nil, err
	}

	ret := []string{}
	for _, node := range order {
		ret = append(ret, node.Name)
	}
	return ret, nil
}

// add a specific dependency to the network, and recurse on the leafs.
func (d dependencies) load(name string, network *topsort.Network) error {
	if network.Get(name) != nil {
		return nil
	}
	network.AddNode(name, nil)

	for _, dep := range d[name] {
		if err := d.load(dep, network); err != nil {
			return err
		}
		if err := network.AddEdge(dep, name); err != nil {
			return err
		}
	}

	return nil
}

//
func loadDependencies() (dependencies, error) {
	path, err := modulePath("modules.dep")
	if err != nil {
		return nil, err
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	deps := map[string][]string{}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		chunks := strings.SplitN(scanner.Text(), ":", 2)
		depString := strings.TrimSpace(chunks[1])
		if len(depString) == 0 {
			continue
		}
		deps[chunks[0]] = strings.Split(depString, " ")
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return deps, nil
}
