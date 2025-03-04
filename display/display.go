package display

import (
	qt "github.com/mappu/miqt/qt6"
)

var UpdateTextEventType = qt.QEvent_RegisterEventType()

type InsertTextEvent struct {
	qt.QEvent
	Text string
}

type Display struct {
	Window *qt.QMainWindow
	Widget *qt.QWidget
	Layout *qt.QGridLayout
	// references to things that will be updated from outside the main thread
	InstrText *qt.QPlainTextEdit // instructions text
}

func NewDisplay() *Display {
	display := new(Display)
	display.Window = qt.NewQMainWindow2()
	display.Window.SetWindowTitle("Goboy")

	// Main widget
	display.Widget = qt.NewQWidget2()
	display.Layout = qt.NewQGridLayout(display.Widget)

	display.Window.SetCentralWidget(display.Widget)
	return display
}

func NewUpdateTextEvent() *qt.QEvent {
	return qt.NewQEvent(qt.QEvent__Type(UpdateTextEventType))
}
