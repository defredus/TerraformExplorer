import { useEffect, useRef } from 'react'
import Editor, { loader, type OnMount } from '@monaco-editor/react'
import * as monaco from 'monaco-editor/editor/editor.api'
import 'monaco-editor/languages/definitions/hcl/register'
import EditorWorker from 'monaco-editor/editor/editor.worker?worker'
import type { SourceRange } from './types'

self.MonacoEnvironment = {
  getWorker: () => new EditorWorker(),
}
loader.config({ monaco })

interface CodeViewerProps {
  content: string
  file: string
  range?: SourceRange
  loading: boolean
}

export function CodeViewer({ content, file, range, loading }: CodeViewerProps) {
  const editorRef = useRef<Parameters<OnMount>[0] | null>(null)
  const monacoRef = useRef<Parameters<OnMount>[1] | null>(null)
  const decorationsRef = useRef<string[]>([])

  const revealRange = () => {
    const editor = editorRef.current
    const monaco = monacoRef.current
    if (!editor || !monaco || !range) return

    const selection = new monaco.Range(
      range.start.line,
      range.start.column,
      range.end.line,
      Math.max(range.end.column, range.start.column + 1),
    )
    decorationsRef.current = editor.deltaDecorations(decorationsRef.current, [
      {
        range: selection,
        options: {
          isWholeLine: true,
          className: 'source-highlight',
          linesDecorationsClassName: 'source-highlight-gutter',
        },
      },
    ])
    editor.revealRangeInCenter(selection)
  }

  const handleMount: OnMount = (editor, monaco) => {
    editorRef.current = editor
    monacoRef.current = monaco
    revealRange()
  }

  useEffect(revealRange, [content, range])

  if (loading) {
    return <div className="code-state"><span className="spinner" /> Reading {file}…</div>
  }
  if (!content) {
    return <div className="code-state">Select an object or file to inspect its HCL.</div>
  }

  return (
    <Editor
      key={file}
      height="100%"
      language="hcl"
      theme="vs-dark"
      value={content}
      onMount={handleMount}
      options={{
        readOnly: true,
        domReadOnly: true,
        minimap: { enabled: false },
        lineNumbersMinChars: 3,
        folding: true,
        fontFamily: 'JetBrains Mono, SFMono-Regular, Consolas, monospace',
        fontSize: 12,
        lineHeight: 20,
        renderLineHighlight: 'none',
        scrollBeyondLastLine: false,
        smoothScrolling: true,
        padding: { top: 12, bottom: 12 },
        overviewRulerBorder: false,
        hideCursorInOverviewRuler: true,
        wordWrap: 'off',
        automaticLayout: true,
      }}
    />
  )
}
