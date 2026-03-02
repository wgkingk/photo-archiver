package gui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"photo-archiver/internal/core/importer"
	"photo-archiver/internal/core/verifier"
	"photo-archiver/internal/storage/sqlite"
)

type App struct {
	store *sqlite.Store
}

func New(store *sqlite.Store) *App {
	return &App{store: store}
}

func (a *App) Run() {
	guiApp := app.New()
	win := guiApp.NewWindow("Photo Archiver")
	win.Resize(fyne.NewSize(1100, 720))

	sourceEntry := widget.NewEntry()
	sourceEntry.SetPlaceHolder("/Volumes/SD_CARD")
	destEntry := widget.NewEntry()
	destEntry.SetPlaceHolder("/Volumes/PHOTO_BACKUP")

	verifySelect := widget.NewSelect([]string{verifier.ModeSize, verifier.ModeHash}, func(string) {})
	verifySelect.SetSelected(verifier.ModeSize)
	dryRunCheck := widget.NewCheck("仅预演（不复制）", nil)
	dryRunCheck.Checked = true

	statusLabel := widget.NewLabel("状态：就绪")
	summaryLabel := widget.NewLabel("等待任务开始")
	progress := widget.NewProgressBarInfinite()
	progress.Hide()

	totalValue := widget.NewLabelWithStyle("--", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	successValue := widget.NewLabelWithStyle("--", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	skippedValue := widget.NewLabelWithStyle("--", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	failedValue := widget.NewLabelWithStyle("--", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	resultText := widget.NewMultiLineEntry()
	resultText.Wrapping = fyne.TextWrapWord
	resultText.Disable()

	historyRows := make([]sqlite.ImportJobSummary, 0, 30)
	headers := []string{"任务ID", "状态", "总数", "成功", "跳过", "失败", "已复制", "开始时间"}
	historyTable := widget.NewTable(
		func() (int, int) { return len(historyRows) + 1, len(headers) },
		func() fyne.CanvasObject {
			label := widget.NewLabel("")
			label.Alignment = fyne.TextAlignLeading
			return label
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			if id.Row == 0 {
				label.SetText(headers[id.Col])
				label.TextStyle = fyne.TextStyle{Bold: true}
				return
			}
			label.TextStyle = fyne.TextStyle{}
			job := historyRows[id.Row-1]
			switch id.Col {
			case 0:
				label.SetText(shortID(job.ID))
			case 1:
				label.SetText(job.Status)
			case 2:
				label.SetText(fmt.Sprintf("%d", job.TotalCount))
			case 3:
				label.SetText(fmt.Sprintf("%d", job.SuccessCount))
			case 4:
				label.SetText(fmt.Sprintf("%d", job.SkippedCount))
			case 5:
				label.SetText(fmt.Sprintf("%d", job.FailedCount))
			case 6:
				label.SetText(formatBytes(job.CopiedBytes))
			default:
				label.SetText(job.StartedAt)
			}
		},
	)
	historyTable.SetColumnWidth(0, 140)
	historyTable.SetColumnWidth(1, 110)
	historyTable.SetColumnWidth(2, 70)
	historyTable.SetColumnWidth(3, 70)
	historyTable.SetColumnWidth(4, 70)
	historyTable.SetColumnWidth(5, 70)
	historyTable.SetColumnWidth(6, 100)
	historyTable.SetColumnWidth(7, 180)

	setRunning := func(running bool) {
		if running {
			progress.Show()
			progress.Start()
			statusLabel.SetText("状态：执行中")
			return
		}
		progress.Stop()
		progress.Hide()
	}

	refreshHistory := func() {
		jobs, err := a.store.ListImportJobs(30)
		if err != nil {
			summaryLabel.SetText("任务记录读取失败")
			return
		}
		historyRows = jobs
		fyne.Do(func() { historyTable.Refresh() })
	}

	var importButton *widget.Button
	importButton = widget.NewButtonWithIcon("开始导入", theme.ConfirmIcon(), func() {
		source := strings.TrimSpace(sourceEntry.Text)
		dest := strings.TrimSpace(destEntry.Text)
		if source == "" || dest == "" {
			statusLabel.SetText("状态：请先选择来源目录和目标目录")
			return
		}
		importButton.Disable()
		setRunning(true)

		go func() {
			res, err := importer.Run(importer.Request{
				SourceRoot: source,
				DestRoot:   dest,
				DryRun:     dryRunCheck.Checked,
				VerifyMode: verifySelect.Selected,
			}, a.store)

			fyne.Do(func() {
				defer importButton.Enable()
				defer setRunning(false)
				if err != nil {
					statusLabel.SetText("状态：导入失败")
					summaryLabel.SetText("任务失败")
					resultText.SetText(err.Error())
					refreshHistory()
					return
				}
				statusLabel.SetText("状态：导入完成")
				summaryLabel.SetText(fmt.Sprintf("总计 %d，成功 %d，跳过 %d，失败 %d", res.TotalCount, res.SuccessCount, res.SkippedCount, res.FailedCount))
				totalValue.SetText(fmt.Sprintf("%d", res.TotalCount))
				successValue.SetText(fmt.Sprintf("%d", res.SuccessCount))
				skippedValue.SetText(fmt.Sprintf("%d", res.SkippedCount))
				failedValue.SetText(fmt.Sprintf("%d", res.FailedCount))
				resultText.SetText(fmt.Sprintf("任务ID: %s\n状态: %s\n总文件数: %d\n成功: %d\n跳过: %d\n失败: %d\n总大小: %s\n已复制: %s", res.JobID, res.Status, res.TotalCount, res.SuccessCount, res.SkippedCount, res.FailedCount, formatBytes(res.TotalBytes), formatBytes(res.CopiedBytes)))
				refreshHistory()
			})
		}()
	})

	sourceBrowseBtn := widget.NewButtonWithIcon("浏览", theme.FolderOpenIcon(), func() {
		d := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil {
				statusLabel.SetText("状态：选择来源目录失败")
				return
			}
			if uri != nil {
				sourceEntry.SetText(uri.Path())
			}
		}, win)
		d.Show()
	})
	destBrowseBtn := widget.NewButtonWithIcon("浏览", theme.FolderOpenIcon(), func() {
		d := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil {
				statusLabel.SetText("状态：选择目标目录失败")
				return
			}
			if uri != nil {
				destEntry.SetText(uri.Path())
			}
		}, win)
		d.Show()
	})

	statsRow := container.NewGridWithColumns(4,
		makeStatBlock("总文件", totalValue),
		makeStatBlock("成功", successValue),
		makeStatBlock("跳过", skippedValue),
		makeStatBlock("失败", failedValue),
	)

	pathForm := widget.NewForm(
		widget.NewFormItem("来源目录", container.NewBorder(nil, nil, nil, sourceBrowseBtn, sourceEntry)),
		widget.NewFormItem("目标目录", container.NewBorder(nil, nil, nil, destBrowseBtn, destEntry)),
		widget.NewFormItem("校验模式", verifySelect),
	)

	importPage := container.NewBorder(
		container.NewVBox(
			widget.NewLabelWithStyle("导入", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewSeparator(),
		),
		nil,
		nil,
		nil,
		container.NewVScroll(container.NewVBox(
			statsRow,
			layout.NewSpacer(),
			pathForm,
			dryRunCheck,
			container.NewHBox(importButton, progress, layout.NewSpacer(), statusLabel),
			summaryLabel,
			widget.NewSeparator(),
			widget.NewLabelWithStyle("任务输出", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			container.NewVScroll(resultText),
		)),
	)

	refreshBtn := widget.NewButtonWithIcon("刷新", theme.ViewRefreshIcon(), refreshHistory)
	historyPage := container.NewBorder(
		container.NewVBox(
			widget.NewLabelWithStyle("任务记录", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewSeparator(),
			container.NewHBox(refreshBtn, layout.NewSpacer(), widget.NewLabel("展示最近 30 条任务")),
		),
		nil,
		nil,
		nil,
		historyTable,
	)

	settingsPage := container.NewBorder(
		container.NewVBox(
			widget.NewLabelWithStyle("设置", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewSeparator(),
		),
		nil,
		nil,
		nil,
		container.NewVBox(
			widget.NewLabel("后续将加入目录模板、并发控制、识别参数。"),
		),
	)

	pages := []fyne.CanvasObject{importPage, historyPage, settingsPage}
	for i := 1; i < len(pages); i++ {
		pages[i].Hide()
	}
	content := container.NewMax(pages...)

	types := []struct {
		Title string
		Icon  fyne.Resource
	}{
		{Title: "导入", Icon: theme.MailSendIcon()},
		{Title: "任务", Icon: theme.HistoryIcon()},
		{Title: "设置", Icon: theme.SettingsIcon()},
	}
	nav := widget.NewList(
		func() int { return len(types) },
		func() fyne.CanvasObject {
			icon := widget.NewIcon(theme.FolderIcon())
			text := widget.NewLabel("")
			return container.NewHBox(icon, text)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			row := obj.(*fyne.Container)
			row.Objects[0].(*widget.Icon).SetResource(types[id].Icon)
			row.Objects[1].(*widget.Label).SetText(types[id].Title)
		},
	)
	nav.OnSelected = func(id widget.ListItemID) {
		for idx := range pages {
			if idx == id {
				pages[idx].Show()
				continue
			}
			pages[idx].Hide()
		}
		content.Refresh()
	}
	nav.Select(0)

	title := widget.NewLabelWithStyle("Photo Archiver", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	subtitle := widget.NewLabel("macOS 风格布局：左侧导航 + 内容区")
	toolbar := container.NewVBox(title, subtitle)

	navPanel := container.NewPadded(container.NewBorder(nil, nil, nil, nil, nav))
	body := container.NewHSplit(navPanel, container.NewPadded(content))
	body.Offset = 0.18

	main := container.NewBorder(toolbar, nil, nil, nil, body)
	refreshHistory()
	win.SetContent(main)
	win.ShowAndRun()
}

func makeStatBlock(title string, value *widget.Label) fyne.CanvasObject {
	label := widget.NewLabel(title)
	label.TextStyle = fyne.TextStyle{Bold: true}
	return container.NewPadded(container.NewVBox(label, value, widget.NewSeparator()))
}

func shortID(id string) string {
	if len(id) <= 10 {
		return id
	}
	return id[:10]
}

func formatBytes(n int64) string {
	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
	)
	switch {
	case n >= gb:
		return fmt.Sprintf("%.2f GB", float64(n)/float64(gb))
	case n >= mb:
		return fmt.Sprintf("%.2f MB", float64(n)/float64(mb))
	case n >= kb:
		return fmt.Sprintf("%.2f KB", float64(n)/float64(kb))
	default:
		return fmt.Sprintf("%d B", n)
	}
}
