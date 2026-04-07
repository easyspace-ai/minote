package notex

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// --- Scan & state ---

type skillFrontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Version     string `yaml:"version"`
}

type scannedSkill struct {
	Slug        string
	Name        string
	Description string
	Version     string
	Source      string // public | custom | workspace
	Root        string
	RelDir      string // path relative to root, posix slashes
}

type skillStateEntry struct {
	Enabled     bool   `json:"enabled"`
	InstalledAt string `json:"installedAt"`
}

type skillsStateFile struct {
	Entries map[string]skillStateEntry `json:"entries"`
}

func (s *Server) skillsStatePath() string {
	return filepath.Join(s.cfg.DataRoot, "skills_state.json")
}

func (s *Server) loadSkillsState() (map[string]skillStateEntry, error) {
	p := s.skillsStatePath()
	b, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]skillStateEntry{}, nil
		}
		return nil, err
	}
	var sf skillsStateFile
	if err := json.Unmarshal(b, &sf); err != nil {
		return nil, err
	}
	if sf.Entries == nil {
		sf.Entries = map[string]skillStateEntry{}
	}
	return sf.Entries, nil
}

func (s *Server) saveSkillsState(m map[string]skillStateEntry) error {
	p := s.skillsStatePath()
	sf := skillsStateFile{Entries: m}
	b, err := json.MarshalIndent(sf, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, b, 0o644)
}

func (s *Server) skillsRoots() []string {
	if len(s.cfg.SkillsPaths) > 0 {
		return s.cfg.SkillsPaths
	}
	seen := make(map[string]struct{})
	var out []string
	add := func(p string) {
		p = filepath.Clean(p)
		if p == "" || p == "." {
			return
		}
		if _, ok := seen[p]; ok {
			return
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	add(filepath.Join(s.cfg.DataRoot, "skills"))
	if cwd, err := os.Getwd(); err == nil {
		add(filepath.Join(cwd, "skills"))
	}
	return out
}

func skillSlugFromRelDir(relDir string) string {
	relDir = filepath.ToSlash(relDir)
	relDir = strings.Trim(relDir, "/")
	if relDir == "" {
		return "skill"
	}
	return strings.ReplaceAll(relDir, "/", "-")
}

func classifySource(relDir string) string {
	rel := filepath.ToSlash(relDir)
	if strings.HasPrefix(rel, "public/") || rel == "public" {
		return "public"
	}
	if strings.HasPrefix(rel, "custom/") || rel == "custom" {
		return "custom"
	}
	return "workspace"
}

func splitYAMLFrontmatter(b []byte) (meta []byte, ok bool) {
	s := strings.TrimSpace(string(b))
	if !strings.HasPrefix(s, "---") {
		return nil, false
	}
	s = strings.TrimPrefix(s, "---")
	s = strings.TrimLeft(s, "\n\r")
	idx := strings.Index(s, "\n---")
	if idx < 0 {
		return nil, false
	}
	meta = []byte(strings.TrimSpace(s[:idx]))
	return meta, true
}

func parseSkillMarkdown(path string) (scannedSkill, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return scannedSkill{}, err
	}
	meta, ok := splitYAMLFrontmatter(b)
	if !ok {
		return scannedSkill{}, fmt.Errorf("no yaml frontmatter")
	}
	var fm skillFrontmatter
	if err := yaml.Unmarshal(meta, &fm); err != nil {
		return scannedSkill{}, err
	}
	fm.Name = strings.TrimSpace(fm.Name)
	fm.Description = strings.TrimSpace(fm.Description)
	fm.Version = strings.TrimSpace(fm.Version)
	if fm.Version == "" {
		fm.Version = "0.0.0"
	}
	return scannedSkill{
		Name:        fm.Name,
		Description: fm.Description,
		Version:     fm.Version,
	}, nil
}

func parseSkillYAML(path string) (scannedSkill, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return scannedSkill{}, err
	}
	var fm skillFrontmatter
	if err := yaml.Unmarshal(b, &fm); err != nil {
		return scannedSkill{}, err
	}
	fm.Name = strings.TrimSpace(fm.Name)
	fm.Description = strings.TrimSpace(fm.Description)
	fm.Version = strings.TrimSpace(fm.Version)
	if fm.Version == "" {
		fm.Version = "0.0.0"
	}
	return scannedSkill{
		Name:        fm.Name,
		Description: fm.Description,
		Version:     fm.Version,
	}, nil
}

func (s *Server) scanInstalledSkills() ([]scannedSkill, error) {
	roots := s.skillsRoots()
	bySlug := map[string]scannedSkill{}
	for _, root := range roots {
		st, err := os.Stat(root)
		if err != nil || !st.IsDir() {
			continue
		}
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				return nil
			}
			// Support both SKILL.md and .yaml files
			isMarkdown := strings.EqualFold(d.Name(), "SKILL.md")
			isYAML := strings.HasSuffix(strings.ToLower(d.Name()), ".yaml") || strings.HasSuffix(strings.ToLower(d.Name()), ".yml")
			if !isMarkdown && !isYAML {
				return nil
			}
			relDir, err := filepath.Rel(root, filepath.Dir(path))
			if err != nil {
				return nil
			}
			var sc scannedSkill
			if isMarkdown {
				parsed, err := parseSkillMarkdown(path)
				if err != nil {
					s.logger.Printf("skills: skip %s: %v", path, err)
					return nil
				}
				sc = parsed
			} else {
				parsed, err := parseSkillYAML(path)
				if err != nil {
					s.logger.Printf("skills: skip %s: %v", path, err)
					return nil
				}
				sc = parsed
			}
			slug := skillSlugFromRelDir(relDir)
			if slug == "" {
				return nil
			}
			// For YAML files in studio folder, use a more specific slug
			if isYAML && relDir != "." {
				baseName := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
				slug = skillSlugFromRelDir(filepath.Join(relDir, baseName))
			}
			if _, dup := bySlug[slug]; dup {
				return nil
			}
			sc.Slug = slug
			sc.Root = root
			sc.RelDir = filepath.ToSlash(relDir)
			sc.Source = classifySource(relDir)
			if sc.Name == "" {
				sc.Name = slug
			}
			bySlug[slug] = sc
			return nil
		})
	}
	out := make([]scannedSkill, 0, len(bySlug))
	for _, v := range bySlug {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Slug < out[j].Slug })
	return out, nil
}

func (s *Server) mergeInstalledDTO() ([]map[string]any, error) {
	scanned, err := s.scanInstalledSkills()
	if err != nil {
		return nil, err
	}
	state, err := s.loadSkillsState()
	if err != nil {
		return nil, err
	}
	now := nowRFC3339()
	modified := false
	out := make([]map[string]any, 0, len(scanned))
	for _, sk := range scanned {
		ent, ok := state[sk.Slug]
		if !ok {
			ent = skillStateEntry{Enabled: true, InstalledAt: now}
			state[sk.Slug] = ent
			modified = true
		} else if strings.TrimSpace(ent.InstalledAt) == "" {
			ent.InstalledAt = now
			state[sk.Slug] = ent
			modified = true
		}
		out = append(out, map[string]any{
			"slug":        sk.Slug,
			"name":        sk.Name,
			"description": sk.Description,
			"version":     sk.Version,
			"source":      sk.Source,
			"enabled":     ent.Enabled,
			"installedAt": ent.InstalledAt,
		})
	}
	if modified {
		if err := s.saveSkillsState(state); err != nil {
			s.logger.Printf("skills: save state: %v", err)
		}
	}
	return out, nil
}

// --- HTTP ---

var skillsPathMu sync.Mutex

func (s *Server) handleSkillsInstalled(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if _, ok := s.requireUserID(w, r); !ok {
		return
	}
	list, err := s.mergeInstalledDTO()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleSkillsWorkspace(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if _, ok := s.requireUserID(w, r); !ok {
		return
	}
	roots := s.skillsRoots()
	dir := ""
	if len(roots) > 0 {
		dir = roots[0]
	}
	writeJSON(w, http.StatusOK, map[string]string{"skills_dir": dir})
}

func (s *Server) handleSkillsRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if _, ok := s.requireUserID(w, r); !ok {
		return
	}
	skillsPathMu.Lock()
	defer skillsPathMu.Unlock()
	list, err := s.mergeInstalledDTO()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleSkillsInstall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if _, ok := s.requireUserID(w, r); !ok {
		return
	}
	var body struct {
		Slug    string `json:"slug"`
		Version string `json:"version"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	slug := strings.TrimSpace(body.Slug)
	if slug == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "slug_required"})
		return
	}
	skillsPathMu.Lock()
	defer skillsPathMu.Unlock()
	scanned, err := s.scanInstalledSkills()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	for _, sk := range scanned {
		if sk.Slug == slug {
			writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
			return
		}
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "skill_not_found_on_disk"})
}

func (s *Server) handleSkillsEnable(w http.ResponseWriter, r *http.Request) {
	s.skillsSetEnabled(w, r, true)
}

func (s *Server) handleSkillsDisable(w http.ResponseWriter, r *http.Request) {
	s.skillsSetEnabled(w, r, false)
}

func (s *Server) skillsSetEnabled(w http.ResponseWriter, r *http.Request, enabled bool) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if _, ok := s.requireUserID(w, r); !ok {
		return
	}
	slug := strings.TrimSpace(r.PathValue("slug"))
	if slug == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_slug"})
		return
	}
	skillsPathMu.Lock()
	defer skillsPathMu.Unlock()
	state, err := s.loadSkillsState()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	scanned, err := s.scanInstalledSkills()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	found := false
	for _, sk := range scanned {
		if sk.Slug == slug {
			found = true
			break
		}
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "skill_not_found"})
		return
	}
	ent, ok := state[slug]
	if !ok {
		ent = skillStateEntry{Enabled: true, InstalledAt: nowRFC3339()}
	}
	ent.Enabled = enabled
	if strings.TrimSpace(ent.InstalledAt) == "" {
		ent.InstalledAt = nowRFC3339()
	}
	state[slug] = ent
	if err := s.saveSkillsState(state); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleSkillsUninstall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if _, ok := s.requireUserID(w, r); !ok {
		return
	}
	slug := strings.TrimSpace(r.PathValue("slug"))
	if slug == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_slug"})
		return
	}
	skillsPathMu.Lock()
	defer skillsPathMu.Unlock()
	scanned, err := s.scanInstalledSkills()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	var target *scannedSkill
	for i := range scanned {
		if scanned[i].Slug == slug {
			target = &scanned[i]
			break
		}
	}
	if target == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "skill_not_found"})
		return
	}
	if !strings.Contains(target.RelDir, "custom/") && !strings.HasPrefix(target.RelDir, "custom") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "only_custom_skills_removable"})
		return
	}
	dir := filepath.Join(target.Root, filepath.FromSlash(target.RelDir))
	if err := os.RemoveAll(dir); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	state, err := s.loadSkillsState()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	delete(state, slug)
	if err := s.saveSkillsState(state); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
