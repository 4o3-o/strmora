package organizer

import (
	"path"
)

type PlannerConfig struct {
	DefaultMovieDir string
	DefaultTVDir    string
	MovieFolderTpl  *Template
	MovieFileTpl    *Template
	TVFolderTpl     *Template
	TVFileTpl       *Template
}

type PlanItem struct {
	ID          string
	CurrentPath string
	Title       string
	Year        int
	Season      int
	Episode     int
	Part        string
	Version     string
	TMDBID      int
	MediaType   string
	Ext         string
}

type Plan struct {
	Item         PlanItem
	TargetPath   string
	OverridePath string
	FolderRel    string
	FileRel      string
}

func (p *Plan) EffectivePath() string {
	if p.OverridePath != "" {
		return p.OverridePath
	}
	return p.TargetPath
}

type Planner struct {
	cfg PlannerConfig
}

func NewPlanner(cfg PlannerConfig) *Planner {
	if cfg.MovieFolderTpl == nil {
		cfg.MovieFolderTpl = NewTemplate("{title} ({year})")
	}
	if cfg.MovieFileTpl == nil {
		cfg.MovieFileTpl = NewTemplate("{title} ({year}).{ext}")
	}
	if cfg.TVFolderTpl == nil {
		cfg.TVFolderTpl = NewTemplate("{title} ({year})")
	}
	if cfg.TVFileTpl == nil {
		cfg.TVFileTpl = NewTemplate("{title} - S{season}E{episode}.{ext}")
	}
	return &Planner{cfg: cfg}
}

func (p *Planner) BuildPlan(it PlanItem) Plan {
	data := TemplateData{
		Title:   it.Title,
		Year:    it.Year,
		Season:  it.Season,
		Episode: it.Episode,
		Part:    it.Part,
		Version: it.Version,
		TMDBID:  it.TMDBID,
		Ext:     it.Ext,
	}

	var folderName string
	var fileName string
	var baseDir string

	if it.MediaType == "series" {
		baseDir = p.cfg.DefaultTVDir
		folderName = p.cfg.TVFolderTpl.Format(data)
		fileName = p.cfg.TVFileTpl.Format(data)
	} else {
		baseDir = p.cfg.DefaultMovieDir
		folderName = p.cfg.MovieFolderTpl.Format(data)
		fileName = p.cfg.MovieFileTpl.Format(data)
	}

	target := path.Join(baseDir, folderName, fileName)
	return Plan{
		Item:       it,
		TargetPath: target,
		FolderRel:  folderName,
		FileRel:    fileName,
	}
}

func (p *Planner) SetOverride(plan *Plan, overrideTargetDir string) {
	plan.OverridePath = path.Join(overrideTargetDir, plan.FolderRel, plan.FileRel)
}
