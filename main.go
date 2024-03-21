package main

import (
	"encoding/json"
	"fmt"
	_ "github.com/ying32/govcl/pkgs/winappres"
	"github.com/ying32/govcl/vcl"
	"github.com/ying32/govcl/vcl/types"
	"github.com/ying32/govcl/vcl/types/colors"
	"github.com/ying32/govcl/vcl/win"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"stream-parser/protocol"
	"strings"
)

type TMainForm struct {
	*vcl.TForm
	MmoInpuFiles *vcl.TMemo
	pnlleft      *vcl.TPanel
	pnlbottom    *vcl.TPanel
	pnlclient    *vcl.TPanel
	mainMenu     *vcl.TMainMenu
	TreeView1    *vcl.TTreeView
	OpenDialog1  *vcl.TOpenDialog
	splitter     *vcl.TSplitter
	splitter2    *vcl.TSplitter

	LvRTP               *vcl.TListView
	LvKeyFrame          *vcl.TListView
	LvVideoPtsJumpFrame *vcl.TListView
	LvAudioPtsJumpFrame *vcl.TListView
	LvVideo             *vcl.TListView
	LvAudio             *vcl.TListView

	subItemHit win.TLVHitTestInfo
}

var (
	mainForm *TMainForm
	result   = protocol.Result{}
)

const (
	rtpRes          = "rtp"
	keyFrameRes     = "keyFrame"
	videoPtsJumpRes = "videoPtsJump"
	audioPtsJumpRes = "audioPtsJump"
	audioVideoRes   = "audioVideo"
	audioRes        = "audioFrame"
	videoRes        = "videoFrame"
)

func main() {
	fmt.Println("mainForm == nil:", mainForm == nil)
	if mainForm != nil {
		mainForm.Free()
	}
	vcl.RunApp(&mainForm)
}

func (f *TMainForm) OnFormCreate(sender vcl.IObject) {

	f.SetCaption("ps流分析工具")
	f.ScreenCenter()
	f.SetWidth(850)
	f.SetHeight(600)

	// ################################ TMainMenu ################################
	f.menuCreate()
	// ##########################################################################################

	// 此时的pnl仅Width属性生效
	pnlleft := vcl.NewPanel(mainForm)
	f.pnlleft = pnlleft
	pnlleft.SetCaption("分析结果导航栏")
	pnlleft.SetParentBackground(false)
	pnlleft.SetColor(colors.ClGrey)
	pnlleft.SetParent(mainForm)
	pnlleft.SetWidth(150)
	pnlleft.SetAlign(types.AlLeft)

	// ##########################################################################################

	// 此时的pnl无法手动调整大小
	pnlclient := vcl.NewPanel(mainForm)
	f.pnlclient = pnlclient
	pnlclient.SetCaption("请点击 文件》打开》选择你要分析的ps流文件开始分析")
	pnlclient.SetParentBackground(false)
	pnlclient.SetParent(mainForm)
	pnlclient.SetAlign(types.AlClient)
	pnlclient.SetAutoSize(true)

	if f.splitter != nil {
		f.splitter.Free()
	}
	f.splitter = vcl.NewSplitter(f)
	f.splitter.SetParent(f)
	f.splitter.SetLeft(f.pnlleft.Width())
}

func (f *TMainForm) showPanelChild(pnlclient *vcl.TPanel, childControlName string) {
	for i := int32(0); i < pnlclient.ControlCount(); i++ {
		controls := pnlclient.Controls(i)
		controls.Hide()
	}
	keyFrameBtn := pnlclient.FindChildControl(childControlName)
	keyFrameBtn.Show()
}

func (f *TMainForm) ListViewRTP(pnlclient *vcl.TPanel) *vcl.TListView {
	lv := vcl.NewListView(pnlclient)
	lv.SetParent(pnlclient)
	lv.SetName("rtp")
	lv.SetAlign(types.AlClient)
	lv.SetRowSelect(true)
	lv.SetReadOnly(true)
	lv.SetViewStyle(types.VsReport)
	lv.SetGridLines(true)
	//lv.SetColumnClick(false)
	lv.SetHideSelection(false)
	lv.SetHeight(500)
	var col *vcl.TListColumn
	t := reflect.TypeOf(protocol.RTPAnalysis{})
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		key := field.Tag.Get("json")
		if key != "-" {
			col = lv.Columns().Add()
			col.SetCaption(key)
			col.SetWidth(100)
		}
	}
	item := lv.Items().Add()
	item.SetCaption(fmt.Sprintf("%d", result.RTPAnalysis.TotalRtp))
	item.SubItems().Add(fmt.Sprintf("%d", result.RTPAnalysis.LostSeqNumber))
	lv.Hide()
	f.SetPopupMenuForLvItem(lv)
	return lv
}

// ============ 关键帧详情  ============
func (f *TMainForm) ListViewKeyFrame(pnlclient *vcl.TPanel) *vcl.TListView {
	lv := vcl.NewListView(pnlclient)
	f.LvKeyFrame = lv
	lv.SetParent(pnlclient)
	lv.SetName(keyFrameRes)
	lv.SetAlign(types.AlClient)
	lv.SetRowSelect(true)
	lv.SetReadOnly(true)
	lv.SetViewStyle(types.VsReport)
	lv.SetGridLines(true)
	//lv.SetColumnClick(false)
	lv.SetHideSelection(false)
	lv.SetHeight(500)

	itemValues := strings.Split((*result.KeyFrame)[0].FrameInfoImportMsg, ",")
	var col *vcl.TListColumn
	for _, str := range itemValues {
		keyVal := strings.Split(strings.TrimSpace(str), ":")
		col = lv.Columns().Add()
		col.SetCaption(keyVal[0])
		col.SetWidth(100)
	}
	// 双击选中项事件
	lv.SetOnDblClick(func(sender vcl.IObject) {
		if lv.ItemIndex() != -1 {
			f.ClearTreeViewItems()
			item := lv.Items().Item(lv.ItemIndex())
			tar, _ := strconv.ParseInt(item.Caption(), 10, 64)
			finfo := result.FrameAnalysis.KeyFrame.FindBinSearchByKey(tar)
			fmt.Println(finfo.IDNumber)
			f.jsonTree(TojsonString(finfo), item.Caption())
			f.TreeView1.SetReadOnly(true)
			f.TreeView1.FullExpand()
		}
	})

	lv.Items().BeginUpdate()

	// 关键帧List item
	for _, info := range *result.KeyFrame {
		itemValues := strings.Split(info.FrameInfoImportMsg, ",")
		item := lv.Items().Add()
		item.SetCaption(fmt.Sprintf("%d", info.IDNumber))
		for _, str := range itemValues {
			keyVal := strings.Split(strings.TrimSpace(str), ":")
			if keyVal[0] != "帧号" {
				item.SubItems().Add(fmt.Sprintf("%s", keyVal[1]))
			}
		}
	}
	lv.Items().EndUpdate()
	lv.Hide()
	f.SetPopupMenuForLvItem(lv)
	return lv
}

// ============ 视频帧跳变详情  ============
func (f *TMainForm) ListViewVideoPtsJumpFrame(pnlclient *vcl.TPanel) *vcl.TListView {
	lv := vcl.NewListView(pnlclient)
	f.LvVideoPtsJumpFrame = lv
	lv.SetParent(pnlclient)
	lv.SetName(videoPtsJumpRes)
	lv.SetAlign(types.AlClient)
	lv.SetRowSelect(true)
	lv.SetReadOnly(true)
	lv.SetViewStyle(types.VsReport)
	lv.SetGridLines(true)
	//lv.SetColumnClick(false)
	lv.SetHideSelection(false)
	lv.SetHeight(500)

	itemValues := strings.Split((*result.JumpDiffFrameVideoList)[0].Msg, ",")
	var col *vcl.TListColumn
	for _, str := range itemValues {
		keyVal := strings.Split(strings.TrimSpace(str), ":")
		col = lv.Columns().Add()
		col.SetCaption(keyVal[0])
		col.SetWidth(100)
	}
	// 双击选中项事件
	lv.SetOnDblClick(func(sender vcl.IObject) {
		if lv.ItemIndex() != -1 {
			f.ClearTreeViewItems()
			item := lv.Items().Item(lv.ItemIndex())
			tar, _ := strconv.ParseInt(item.Caption(), 10, 64)
			finfo := result.FrameAnalysis.JumpDiffFrameVideoList.FindBinSearchByKey(tar)
			f.jsonTree(TojsonString(finfo), item.Caption())
			f.TreeView1.FullExpand()
			f.TreeView1.SetReadOnly(true)
		}
	})
	lv.Items().BeginUpdate()

	for _, info := range *result.JumpDiffFrameVideoList {
		itemValues := strings.Split(info.Msg, ",")
		item := lv.Items().Add()
		item.SetCaption(fmt.Sprintf("%d", info.IDNumber))
		for _, str := range itemValues {
			keyVal := strings.Split(strings.TrimSpace(str), ":")
			if keyVal[0] != "帧号" {
				item.SubItems().Add(fmt.Sprintf("%s", keyVal[1]))
			}
		}
	}
	lv.Items().EndUpdate()
	lv.Hide()
	f.SetPopupMenuForLvItem(lv)
	return lv
}

// ============ 音频帧跳变详情  ============
func (f *TMainForm) ListViewAudioPtsJumpFrame(pnlclient *vcl.TPanel) *vcl.TListView {
	lv := vcl.NewListView(pnlclient)
	f.LvAudioPtsJumpFrame = lv
	lv.SetParent(pnlclient)
	lv.SetName(audioPtsJumpRes)
	lv.SetAlign(types.AlClient)
	lv.SetRowSelect(true)
	lv.SetReadOnly(true)
	lv.SetViewStyle(types.VsReport)
	lv.SetGridLines(true)
	//lv.SetColumnClick(false)
	lv.SetHideSelection(false)
	lv.SetHeight(500)

	itemValues := strings.Split((*result.JumpDiffFrameVideoList)[0].Msg, ",")
	var col *vcl.TListColumn
	for _, str := range itemValues {
		keyVal := strings.Split(strings.TrimSpace(str), ":")
		col = lv.Columns().Add()
		col.SetCaption(keyVal[0])
		col.SetWidth(100)
	}
	// 双击选中项事件
	lv.SetOnDblClick(func(sender vcl.IObject) {
		if lv.ItemIndex() != -1 {
			f.ClearTreeViewItems()
			item := lv.Items().Item(lv.ItemIndex())
			tar, _ := strconv.ParseInt(item.Caption(), 10, 64)
			finfo := result.FrameAnalysis.JumpDiffFrameAudioList.FindBinSearchByKey(tar)
			fmt.Println(finfo.IDNumber)
			f.jsonTree(TojsonString(finfo), item.Caption())
			f.TreeView1.FullExpand()
			f.TreeView1.SetReadOnly(true)
		}
	})
	lv.Items().BeginUpdate()

	for _, info := range *result.JumpDiffFrameAudioList {
		itemValues := strings.Split(info.Msg, ",")
		item := lv.Items().Add()
		item.SetCaption(fmt.Sprintf("%d", info.IDNumber))
		for _, str := range itemValues {
			keyVal := strings.Split(strings.TrimSpace(str), ":")
			if keyVal[0] != "帧号" {
				item.SubItems().Add(fmt.Sprintf("%s", keyVal[1]))
			}
		}
	}
	lv.Items().EndUpdate()
	lv.Hide()
	f.SetPopupMenuForLvItem(lv)
	return lv
}

// ============ 音频视频分布详情  ============
func (f *TMainForm) ListViewAudioVideo(pnlclient *vcl.TPanel) *vcl.TListView {
	lv := vcl.NewListView(pnlclient)
	f.LvAudio = lv
	lv.SetParent(pnlclient)
	lv.SetName(audioVideoRes)
	lv.SetAlign(types.AlClient)
	lv.SetRowSelect(true)
	lv.SetReadOnly(true)
	lv.SetViewStyle(types.VsReport)
	lv.SetGridLines(true)
	//lv.SetColumnClick(false)
	lv.SetHideSelection(false)
	lv.SetHeight(500)

	col := lv.Columns().Add()
	col.SetCaption("音频帧视频帧分布")
	col.SetWidth(100)

	lv.Items().BeginUpdate()
	for _, msg := range (*result.FrameAnalysis).FrameNumberString {
		item := lv.Items().Add()
		item.SetCaption(fmt.Sprintf("%s", msg))
	}
	lv.Items().EndUpdate()
	lv.Hide()
	f.SetPopupMenuForLvItem(lv)
	return lv
}

// ============ 视频详情  ============
func (f *TMainForm) ListViewVideo(pnlclient *vcl.TPanel) *vcl.TListView {
	lv := vcl.NewListView(pnlclient)
	f.LvVideo = lv
	lv.SetParent(pnlclient)
	lv.SetName(videoRes)
	lv.SetAlign(types.AlClient)
	lv.SetRowSelect(true)
	lv.SetReadOnly(true)
	lv.SetViewStyle(types.VsReport)
	lv.SetGridLines(true)
	//lv.SetColumnClick(false)
	lv.SetHideSelection(false)
	lv.SetHeight(500)

	itemValues := strings.Split((*result.VideoFrame)[0].FrameInfoImportMsg, ",")
	var col *vcl.TListColumn
	for _, str := range itemValues {
		keyVal := strings.Split(strings.TrimSpace(str), ":")
		col = lv.Columns().Add()
		col.SetCaption(keyVal[0])
		col.SetWidth(100)
	}
	// 双击选中项事件
	lv.SetOnDblClick(func(sender vcl.IObject) {
		if lv.ItemIndex() != -1 {
			f.ClearTreeViewItems()
			item := lv.Items().Item(lv.ItemIndex())
			tar, _ := strconv.ParseInt(item.Caption(), 10, 64)
			finfo := result.FrameAnalysis.VideoFrame.FindBinSearchByKey(tar)
			fmt.Println(finfo.IDNumber)
			f.jsonTree(TojsonString(finfo), item.Caption())
			f.TreeView1.SetReadOnly(true)
			f.TreeView1.FullExpand()
		}
	})
	lv.Items().BeginUpdate()

	for _, info := range *result.VideoFrame {
		itemValues := strings.Split(info.FrameInfoImportMsg, ",")
		item := lv.Items().Add()
		item.SetCaption(fmt.Sprintf("%d", info.IDNumber))
		for _, str := range itemValues {
			keyVal := strings.Split(strings.TrimSpace(str), ":")
			if keyVal[0] != "帧号" {
				if (keyVal[0] == "pts差" || keyVal[0] == "上一帧的pts") && keyVal[1] == "0" {
					item.SubItems().Add("-")
				} else {
					item.SubItems().Add(fmt.Sprintf("%s", keyVal[1]))
				}
			}
		}
	}
	lv.SetOnAdvancedCustomDrawSubItem(func(sender *vcl.TListView, item *vcl.TListItem, subItem int32, state types.TCustomDrawState, stage types.TCustomDrawStage, defaultDraw *bool) {
		s := item.SubItems().S(subItem - 1)
		if subItem == 1 && s == "true" { // 是否关键帧
			sender.Canvas().Brush().SetColor(0x02F0EEF7)
		}
		if subItem == 6 && s == "true" { // 是否跳变
			sender.Canvas().Brush().SetColor(colors.ClRed)
		}
	})
	//lv.SetOnAdvancedCustomDrawItem(func(sender *vcl.TListView, item *vcl.TListItem, state types.TCustomDrawState, Stage types.TCustomDrawStage, defaultDraw *bool) {
	//
	//	s := item.SubItems().S(subItem - 1)
	//	if subItem == 2 && s == "true" {
	//		fmt.Println("subItem:", subItem, "s:", s)
	//		sender.Canvas().Brush().SetColor(0x02F0EEF7)
	//		//canvas.Brush().SetColor(0x02F0EEF7)
	//	}
	//
	//})

	lv.Items().EndUpdate()
	lv.Hide()
	f.SetPopupMenuForLvItem(lv)
	return lv
}

// ============ 音频详情  ============
func (f *TMainForm) ListViewAudio(pnlclient *vcl.TPanel) *vcl.TListView {
	lv := vcl.NewListView(pnlclient)
	f.LvAudio = lv
	lv.SetParent(pnlclient)
	lv.SetName(audioRes)
	lv.SetAlign(types.AlClient)
	lv.SetRowSelect(true)
	lv.SetReadOnly(true)
	lv.SetViewStyle(types.VsReport)
	lv.SetGridLines(true)
	//lv.SetColumnClick(false)
	lv.SetHideSelection(false)
	lv.SetHeight(500)

	itemValues := strings.Split((*result.AudioFrame)[0].FrameInfoImportMsg, ",")
	var col *vcl.TListColumn
	for _, str := range itemValues {
		keyVal := strings.Split(strings.TrimSpace(str), ":")
		col = lv.Columns().Add()
		col.SetCaption(keyVal[0])
		col.SetWidth(100)
	}
	// 双击选中项事件
	lv.SetOnDblClick(func(sender vcl.IObject) {
		if lv.ItemIndex() != -1 {
			f.ClearTreeViewItems()
			item := lv.Items().Item(lv.ItemIndex())
			tar, _ := strconv.ParseInt(item.Caption(), 10, 64)
			finfo := result.FrameAnalysis.AudioFrame.FindBinSearchByKey(tar)
			fmt.Println(finfo.IDNumber)
			f.jsonTree(TojsonString(finfo), item.Caption())
			f.TreeView1.FullExpand()
			f.TreeView1.SetReadOnly(true)
		}
	})
	lv.Items().BeginUpdate()

	for _, info := range *result.AudioFrame {
		itemValues := strings.Split(info.FrameInfoImportMsg, ",")
		item := lv.Items().Add()
		item.SetCaption(fmt.Sprintf("%d", info.IDNumber))
		for _, str := range itemValues {
			keyVal := strings.Split(strings.TrimSpace(str), ":")
			if keyVal[0] != "帧号" {
				if (keyVal[0] == "pts差" || keyVal[0] == "上一帧的pts") && keyVal[1] == "0" {
					item.SubItems().Add("-")
				} else {
					item.SubItems().Add(fmt.Sprintf("%s", keyVal[1]))
				}
			}
		}
	}
	lv.Items().EndUpdate()
	lv.SetOnAdvancedCustomDrawSubItem(func(sender *vcl.TListView, item *vcl.TListItem, subItem int32, state types.TCustomDrawState, stage types.TCustomDrawStage, defaultDraw *bool) {
		s := item.SubItems().S(subItem - 1)
		if subItem == 1 && s == "true" { // 是否关键帧
			sender.Canvas().Brush().SetColor(0x02F0EEF7)
		}
		if subItem == 5 && s == "true" { // 是否跳变
			sender.Canvas().Brush().SetColor(colors.ClRed)
		}
	})

	lv.Hide()
	f.SetPopupMenuForLvItem(lv)
	return lv
}

func (f *TMainForm) menuCreate() {
	f.mainMenu = vcl.NewMainMenu(f)
	f.mainMenu.SetOnMeasureItem(func(sender vcl.IObject, aCanvas *vcl.TCanvas, width, height *int32) {
		*height = 44
	})

	// 一级菜单
	item := vcl.NewMenuItem(f)
	item.SetCaption("文件(&F)")

	subMenu := vcl.NewMenuItem(f)
	subMenu.SetCaption("新建(&N)")
	subMenu.SetShortCutFromString("Ctrl+N")
	subMenu.SetOnClick(func(vcl.IObject) {
		fmt.Println("单击了新建")
	})
	item.Add(subMenu)

	subMenuOpen := vcl.NewMenuItem(f)
	subMenuOpen.SetCaption("打开(&O)")
	subMenuOpen.SetShortCutFromString("Ctrl+O")
	item.Add(subMenuOpen)

	// 弹窗
	dlgOpen := vcl.NewOpenDialog(mainForm)
	f.OpenDialog1 = dlgOpen
	dlgOpen.SetFilter("文本文件(*.ps)|*.ps|所有文件(*.*)|*.*")
	dlgOpen.SetOptions(dlgOpen.Options().Include(types.OfShowHelp, types.OfAllowMultiSelect)) //rtl.Include(, types.OfShowHelp))
	dlgOpen.SetTitle("Open existing file")
	// 设置事件
	subMenuOpen.SetOnClick(func(sender vcl.IObject) {
		if f.OpenDialog1.Execute() {
			fileName := f.OpenDialog1.FileName()
			f.HandleTOGenerateJson(fileName)
		}
	})

	subMenu = vcl.NewMenuItem(f)
	subMenu.SetCaption("保存(&S)")
	subMenu.SetShortCutFromString("Ctrl+S")
	item.Add(subMenu)

	// 分割线
	subMenu = vcl.NewMenuItem(f)
	subMenu.SetCaption("-")
	item.Add(subMenu)

	/*subMenu = vcl.NewMenuItem(f)
	subMenu.SetCaption("历史记录...")
	item.Add(subMenu)

	m := vcl.NewMenuItem(f)
	m.SetCaption("第一个历史记录")
	subMenu.Add(m)

	subMenu = vcl.NewMenuItem(f)
	subMenu.SetCaption("-")
	item.Add(subMenu)*/

	subMenu = vcl.NewMenuItem(f)
	subMenu.SetCaption("退出(&Q)")
	subMenu.SetShortCutFromString("Ctrl+Q")
	subMenu.SetOnClick(func(vcl.IObject) {
		f.Close()
	})
	item.Add(subMenu)

	f.mainMenu.Items().Add(item)

	item = vcl.NewMenuItem(f)
	item.SetCaption("关于(&A)")

	subMenu = vcl.NewMenuItem(f)
	subMenu.SetCaption("帮助(&H)")
	item.Add(subMenu)
	f.mainMenu.Items().Add(item)

	// TPopupMenu
	pm := vcl.NewPopupMenu(f)
	item = vcl.NewMenuItem(f)
	item.SetCaption("退出(&E)")
	item.SetOnClick(func(vcl.IObject) {
		f.Close()
	})
	pm.Items().Add(item)
	//
	//// 将窗口设置一个弹出菜单，右键单击就可显示
	//f.SetPopupMenu(pm)
}

func (f *TMainForm) SetPopupMenuForLvItem(lv *vcl.TListView) {
	popupMenu := vcl.NewPopupMenu(f)
	itemCopy := vcl.NewMenuItem(f)
	itemCopy.SetCaption("复制")
	itemCopy.SetOnClick(func(vcl.IObject) {
		f.Close()
	})
	popupMenu.Items().Add(itemCopy)

	// 绑定ListView的OnContextPopup事件，显示右键菜单
	lv.SetOnContextPopup(func(sender vcl.IObject, mousePos types.TPoint, handled *bool) {
		// 将鼠标位置转换为屏幕坐标
		mousePos = lv.ClientToScreen(mousePos)
		popupMenu.Popup(mousePos.X, mousePos.Y)
	})

	// 绑定右键菜单的OnClick事件，实现复制功能
	itemCopy.SetOnClick(func(sender vcl.IObject) {
		p := f.LvVideo.ScreenToClient(vcl.Mouse.CursorPos())
		f.subItemHit.Pt.X = p.X
		f.subItemHit.Pt.Y = p.Y
		win.ListView_SubItemHitTest(f.LvVideo.Handle(), &f.subItemHit)
		if f.subItemHit.IItem != -1 {
			if f.subItemHit.ISubItem > 0 {
				content := lv.Selected().SubItems().Strings(f.subItemHit.ISubItem - 1)
				vcl.Clipboard.SetTextBuf(content)
			}
		}
	})
}

func (f *TMainForm) OnFormPaint(sender vcl.IObject) {
	///r := types.TRect{0, 0, f.Width(), f.Height()}
	//f.Canvas().TextRect(r, 0, 0, "右键弹出菜单")
	f.Canvas().Brush().SetStyle(types.BsClear)
	f.Canvas().Font().SetColor(colors.ClGreen)
	f.Canvas().TextOut(10, 80, "右键弹出菜单")
}

func TojsonString(i interface{}) string {
	marshal, _ := json.Marshal(i)
	return string(marshal)
}

func (f *TMainForm) HandleTOGenerateJson(filepath string) {
	executable := "./parser-gen-json.exe"
	cmd := exec.Command(executable, "-file", filepath)

	fmt.Println(cmd.String())
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("Error executing command:", err)
		return
	}
	fmt.Println("output:", string(output))

	file, err := os.ReadFile("./res.json")
	if err := json.Unmarshal(file, &result); err != nil {
		vcl.ShowMessage(err.Error())
		return
	}
	f.AfterParseForm()
}

func (f *TMainForm) AfterParseForm() {
	// 此时的pnl仅Width属性生效
	if f.pnlleft != nil {
		f.pnlleft.Free()
	}
	pnlleft := vcl.NewPanel(mainForm)
	f.pnlleft = pnlleft
	pnlleft.SetCaption("分析结果")
	pnlleft.SetParentBackground(false)
	pnlleft.SetColor(colors.ClGrey)
	pnlleft.SetParent(mainForm)
	pnlleft.SetParent(mainForm)
	pnlleft.SetWidth(150)
	pnlleft.SetAlign(types.AlLeft)

	btnRTPRes := vcl.NewButton(mainForm)
	btnRTPRes.SetAlign(types.AlTop)
	btnRTPRes.SetParent(pnlleft)
	btnRTPRes.SetCaption("RTP分析结果")

	//btnAudioPtsJumpRes := vcl.NewButton(mainForm)
	//btnAudioPtsJumpRes.SetAlign(types.AlTop)
	//btnAudioPtsJumpRes.SetParent(pnlleft)
	//btnAudioPtsJumpRes.SetCaption("音频帧跳变分析结果")

	btnVideoPtsJump := vcl.NewButton(mainForm)
	btnVideoPtsJump.SetAlign(types.AlTop)
	btnVideoPtsJump.SetParent(pnlleft)
	btnVideoPtsJump.SetCaption("视频帧跳变分析结果")

	btnVideoAudioJump := vcl.NewButton(mainForm)
	btnVideoAudioJump.SetAlign(types.AlTop)
	btnVideoAudioJump.SetParent(pnlleft)
	btnVideoAudioJump.SetCaption("视频音频分布分析结果")

	btnKeyFrameRes := vcl.NewButton(mainForm)
	btnKeyFrameRes.SetAlign(types.AlTop)
	btnKeyFrameRes.SetParent(pnlleft)
	btnKeyFrameRes.SetCaption("关键帧")

	btnAudioFrameRes := vcl.NewButton(mainForm)
	btnAudioFrameRes.SetAlign(types.AlTop)
	btnAudioFrameRes.SetParent(pnlleft)
	btnAudioFrameRes.SetCaption("音频帧")

	btnVideoFrameRes := vcl.NewButton(mainForm)
	btnVideoFrameRes.SetAlign(types.AlTop)
	btnVideoFrameRes.SetParent(pnlleft)
	btnVideoFrameRes.SetCaption("视频帧")

	// ##########################################################################################

	// 此时的pnl无法手动调整大小
	if f.pnlclient != nil {
		f.pnlclient.Free()
	}
	pnlclient := vcl.NewPanel(mainForm)
	f.pnlclient = pnlclient
	pnlclient.SetCaption("请点击左侧栏选择要查看的内容")
	pnlclient.SetParentBackground(false)
	pnlclient.SetParent(mainForm)
	pnlclient.SetAlign(types.AlClient)

	// ##########################################################################################

	// 此时的pnl仅Height属性生效
	if f.pnlbottom != nil {
		f.pnlbottom.Free()
	}
	pnlbottom := vcl.NewPanel(mainForm)
	f.pnlbottom = pnlbottom
	pnlbottom.SetCaption("bottom")
	pnlbottom.SetParentBackground(false)
	pnlbottom.SetColor(colors.ClNavajowhite)
	pnlbottom.SetParent(mainForm)
	pnlbottom.SetHeight(200)
	pnlbottom.SetAlign(types.AlBottom)

	f.TreeView1 = vcl.NewTreeView(pnlbottom)
	f.TreeView1.SetParent(pnlbottom)
	f.TreeView1.SetAlign(types.AlClient)

	f.splitter2 = vcl.NewSplitter(f)
	f.splitter2.SetParent(f)
	f.splitter2.SetTop(f.pnlleft.Height())
	f.splitter2.SetAlign(types.AlBottom)

	// ######################################ListView ####################################################

	f.ListViewRTP(pnlclient)
	f.ListViewKeyFrame(pnlclient)
	f.ListViewVideoPtsJumpFrame(pnlclient)
	//f.ListViewAudioPtsJumpFrame(pnlclient)
	f.ListViewAudioVideo(pnlclient)
	f.ListViewAudio(pnlclient)
	f.ListViewVideo(pnlclient)

	// ##########################################################################################

	btnKeyFrameRes.SetOnClick(func(sender vcl.IObject) {
		f.showPanelChild(pnlclient, keyFrameRes)
	})

	btnVideoPtsJump.SetOnClick(func(sender vcl.IObject) {
		f.showPanelChild(pnlclient, videoPtsJumpRes)
	})

	//btnAudioPtsJumpRes.SetOnClick(func(sender vcl.IObject) {
	//	f.showPanelChild(pnlclient, audioPtsJumpRes)
	//})

	btnVideoFrameRes.SetOnClick(func(sender vcl.IObject) {
		f.showPanelChild(pnlclient, videoRes)
	})

	btnAudioFrameRes.SetOnClick(func(sender vcl.IObject) {
		f.showPanelChild(pnlclient, audioRes)
	})

	btnRTPRes.SetOnClick(func(sender vcl.IObject) {
		f.showPanelChild(pnlclient, rtpRes)
	})

	btnVideoAudioJump.SetOnClick(func(sender vcl.IObject) {
		f.showPanelChild(pnlclient, audioVideoRes)
	})

	//===========================================
}
