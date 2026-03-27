package config

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "github.com/mgoulish/mentat/internal/debug"
    "github.com/mgoulish/mentat/internal/new"
)

func ReadNetwork(mentat *new.Mentat) {
    debug.Info(fmt.Sprintf("Reading network from root: %s", mentat.Root))

    entries, err := os.ReadDir(mentat.Root)
    if err != nil {
        debug.Info(fmt.Sprintf("Failed to read root dir: %v", err))
        return
    }

    for _, entry := range entries {
        if entry.IsDir() {
            sitePath := filepath.Join(mentat.Root, entry.Name())
            site := readSite(sitePath)
            if site.Name != "" {
                mentat.Sites = append(mentat.Sites, site)
                debug.Info(fmt.Sprintf("Added site: %s", site.Name))
            }
        }
    }

    getSiteRouters(mentat)
    debug.Info(fmt.Sprintf("Discovered %d sites", len(mentat.Sites)))
}

func readSite(path string) new.Site {
    siteName := filepath.Base(path)
    site := new.NewSite(siteName, path)

    configDir := filepath.Join(path, "configmaps")
    siteYaml := filepath.Join(configDir, "skupper-site.yaml")

    if data, err := os.ReadFile(siteYaml); err == nil {
        content := string(data)
        if idx := strings.Index(content, "ingress-host:"); idx != -1 {
            rest := content[idx+13:]
            if end := strings.IndexAny(rest, "\n "); end != -1 {
                site.IngressHost = strings.TrimSpace(rest[:end])
            }
        }
    }

    return site
}

func getSiteRouters(mentat *new.Mentat) {
    for i := range mentat.Sites {
        site := &mentat.Sites[i]
        podsDir := filepath.Join(site.Root, "pods")

        entries, err := os.ReadDir(podsDir)
        if err != nil {
            continue
        }

        for _, entry := range entries {
            if entry.IsDir() && strings.HasPrefix(entry.Name(), "skupper-router-") {
                routerName := entry.Name()
                nickname := strings.TrimPrefix(routerName, "skupper-router-")
                router := new.NewRouter(routerName, site.Name, nickname)
                site.Routers = append(site.Routers, router)
                debug.Info(fmt.Sprintf("Added router %s to site %s", nickname, site.Name))
            }
        }
    }
}
