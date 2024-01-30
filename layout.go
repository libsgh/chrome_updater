package main

import "fyne.io/fyne/v2"

type buttonLayout struct{}

func (l *buttonLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	// 将选项卡容器放在顶部，按钮放在底部
	tabContent := objects[0]
	button := objects[1]

	buttonSize := button.MinSize()
	button.Resize(fyne.NewSize(size.Width, buttonSize.Height))
	button.Move(fyne.NewPos(0, size.Height-buttonSize.Height))

	tabContent.Resize(fyne.NewSize(size.Width, size.Height-buttonSize.Height-20))
	tabContent.Move(fyne.NewPos(0, 0))
}

func (l *buttonLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	// 计算布局的最小尺寸
	tabContent := objects[0]
	button := objects[1]

	tabContentMinSize := tabContent.MinSize()
	buttonMinSize := button.MinSize()

	return fyne.NewSize(fyne.Max(tabContentMinSize.Width, buttonMinSize.Width),
		tabContentMinSize.Height+buttonMinSize.Height+20)
}
