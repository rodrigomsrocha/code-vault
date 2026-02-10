package ui

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rodrigomsrocha/code-vault/internal/models"
)

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	previewStyle      = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("62")).
				Padding(1, 2).
				MarginLeft(2)
)

// Mensagens customizadas
type previewLoadedMsg struct {
	snippetID string
	content   string
}

type preloadProgressMsg struct {
	loaded int
	total  int
}

type SnippetItem struct {
	Snippet models.Snippet
	Content string
}

func (i SnippetItem) FilterValue() string {
	return i.Snippet.Title
}

type ItemDelegate struct{}

func (d ItemDelegate) Height() int                               { return 3 }
func (d ItemDelegate) Spacing() int                              { return 1 }
func (d ItemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d ItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(SnippetItem)
	if !ok {
		return
	}

	snippet := i.Snippet
	str := fmt.Sprintf("%s", snippet.Title)

	var metadata []string
	metadata = append(metadata, snippet.Language)

	if len(snippet.Tags) > 0 {
		var tagNames []string
		for _, tag := range snippet.Tags {
			tagNames = append(tagNames, tag.Name)
		}
		metadata = append(metadata, strings.Join(tagNames, ", "))
	}

	if snippet.CurrentVersion != nil {
		size := fmt.Sprintf("%d bytes", snippet.CurrentVersion.SizeBytes)
		metadata = append(metadata, size)
	}

	metaStr := strings.Join(metadata, " • ")

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("→ " + strings.Join(s, " "))
		}
		str = fmt.Sprintf("%s\n  %s", str, metaStr)
	} else {
		str = fmt.Sprintf("%s\n  %s", str, metaStr)
	}

	fmt.Fprint(w, fn(str))
}

type keyMap struct {
	Up      key.Binding
	Down    key.Binding
	Enter   key.Binding
	Quit    key.Binding
	Filter  key.Binding
	Preview key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "esc", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		Preview: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "toggle preview"),
		),
	}
}

type ListModel struct {
	list             list.Model
	viewport         viewport.Model
	keys             keyMap
	choice           *models.Snippet
	quitting         bool
	showPreview      bool
	width            int
	height           int
	contentLoader    func(snippetID string) (string, error)
	previewCache     map[string]string
	preloadedCount   int
	totalSnippets    int
	currentSnippetID string
}

func NewListModel(snippets []models.Snippet, contentLoader func(snippetID string) (string, error)) ListModel {
	items := make([]list.Item, len(snippets))
	for i, s := range snippets {
		items[i] = SnippetItem{Snippet: s}
	}

	const defaultWidth = 80
	const listHeight = 20

	l := list.New(items, ItemDelegate{}, defaultWidth, listHeight)
	l.Title = "📚 Select a Snippet"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	vp := viewport.New(40, listHeight-2)
	vp.SetContent("⏳ Loading preview...")

	return ListModel{
		list:          l,
		viewport:      vp,
		keys:          newKeyMap(),
		showPreview:   true,
		contentLoader: contentLoader,
		previewCache:  make(map[string]string),
		totalSnippets: len(snippets),
	}
}

func (m ListModel) Init() tea.Cmd {
	return tea.Batch(
		m.loadCurrentPreview(),
		m.preloadAllPreviewsAsync(),
	)
}

func (m ListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case previewLoadedMsg:
		// Atualiza cache
		m.previewCache[msg.snippetID] = msg.content

		// Se é o snippet atual, mostra no viewport
		if msg.snippetID == m.currentSnippetID {
			m.viewport.SetContent(msg.content)
		}
		return m, nil

	case preloadProgressMsg:
		m.preloadedCount = msg.loaded
		// Opcional: atualizar título com progresso
		// m.list.Title = fmt.Sprintf("📚 Select a Snippet (loaded %d/%d)", msg.loaded, msg.total)
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if m.showPreview {
			listWidth := (msg.Width*2)/5 - 1
			previewWidth := (msg.Width*3)/5 - 2

			m.list.SetWidth(listWidth)
			m.list.SetHeight(msg.Height - 4)
			m.viewport.Width = previewWidth
			m.viewport.Height = msg.Height - 6
		} else {
			m.list.SetWidth(msg.Width)
			m.list.SetHeight(msg.Height - 4)
		}

		return m, m.loadCurrentPreview()

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			i, ok := m.list.SelectedItem().(SnippetItem)
			if ok {
				m.choice = &i.Snippet
			}
			return m, tea.Quit

		case "p":
			m.showPreview = !m.showPreview
			return m, func() tea.Msg {
				return tea.WindowSizeMsg{
					Width:  m.width,
					Height: m.height,
				}
			}
		}
	}

	var cmd tea.Cmd
	var cmds []tea.Cmd

	prevIndex := m.list.Index()
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	// Se mudou de seleção, carrega novo preview
	if m.showPreview && prevIndex != m.list.Index() {
		cmds = append(cmds, m.loadCurrentPreview())
	}

	if m.showPreview {
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// loadCurrentPreview carrega o preview do snippet selecionado
func (m *ListModel) loadCurrentPreview() tea.Cmd {
	if !m.showPreview || m.contentLoader == nil {
		return nil
	}

	item := m.list.SelectedItem()
	if item == nil {
		return nil
	}

	snippetItem, ok := item.(SnippetItem)
	if !ok {
		return nil
	}

	snippetID := snippetItem.Snippet.ID.String()
	m.currentSnippetID = snippetID

	// Se já está no cache, usa imediatamente
	if cached, ok := m.previewCache[snippetID]; ok {
		m.viewport.SetContent(cached)
		return nil
	}

	// Mostra loading
	m.viewport.SetContent("⏳ Loading preview...")

	// Carrega assíncronamente
	return func() tea.Msg {
		content, err := m.contentLoader(snippetID)
		if err != nil {
			return previewLoadedMsg{
				snippetID: snippetID,
				content:   fmt.Sprintf("❌ Error: %v", err),
			}
		}

		// Trunca se necessário
		if len(content) > 2000 {
			content = content[:2000] + "\n\n... (truncated)"
		}

		// Syntax highlighting
		highlighted, err := HighlightCode(content, snippetItem.Snippet.Language)
		if err == nil {
			content = highlighted
		}

		return previewLoadedMsg{
			snippetID: snippetID,
			content:   content,
		}
	}
}

// preloadAllPreviewsAsync carrega todos os previews em background
func (m *ListModel) preloadAllPreviewsAsync() tea.Cmd {
	if m.contentLoader == nil {
		return nil
	}

	return func() tea.Msg {
		items := m.list.Items()
		loaded := 0

		for i := 0; i < len(items); i++ {
			snippetItem, ok := items[i].(SnippetItem)
			if !ok {
				continue
			}

			snippetID := snippetItem.Snippet.ID.String()

			// Pula se já está no cache
			if _, ok := m.previewCache[snippetID]; ok {
				loaded++
				continue
			}

			// Pequeno delay para não sobrecarregar
			time.Sleep(50 * time.Millisecond)

			content, err := m.contentLoader(snippetID)
			if err != nil {
				continue
			}

			if len(content) > 2000 {
				content = content[:2000] + "\n\n... (truncated)"
			}

			highlighted, err := HighlightCode(content, snippetItem.Snippet.Language)
			if err == nil {
				m.previewCache[snippetID] = highlighted
			} else {
				m.previewCache[snippetID] = content
			}

			loaded++
		}

		return preloadProgressMsg{
			loaded: loaded,
			total:  len(items),
		}
	}
}

func (m ListModel) View() string {
	if m.choice != nil {
		return ""
	}
	if m.quitting {
		return ""
	}

	if !m.showPreview {
		return "\n" + m.list.View()
	}

	listView := m.list.View()
	previewView := previewStyle.Render(m.viewport.View())

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		listView,
		previewView,
	)
}

func (m ListModel) GetChoice() *models.Snippet {
	return m.choice
}
