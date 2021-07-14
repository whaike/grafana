import React, { useRef } from 'react';
import { HistoryItem } from '@grafana/data';
import { CodeEditor } from '@grafana/ui';
import { PromQuery } from '../../types';
import type * as monacoType from 'monaco-editor/esm/vs/editor/editor.api';
import type PromQlLanguageProvider from '../../language_provider';
import { useLatest } from 'react-use';
import { setupPromQL } from './setupPromQL';

type PromHistoryItem = HistoryItem<PromQuery>;

const options: monacoType.editor.IStandaloneEditorConstructionOptions = {
  lineNumbers: 'off',
  minimap: { enabled: false },
  lineDecorationsWidth: 0,
  wordWrap: 'off',
  overviewRulerLanes: 0,
  overviewRulerBorder: false,
  folding: false,
  scrollBeyondLastLine: false,
  // FIXME: more might be needed
};

type Props = {
  initialValue: string;
  languageProvider: PromQlLanguageProvider;
  history: PromHistoryItem[];
  onChange: (query: string) => void;
};

export const MonacoQueryField = (props: Props) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const { languageProvider, history, onChange, initialValue } = props;

  const lpRef = useLatest(languageProvider);
  const historyRef = useLatest(history);

  return (
    <div
      // NOTE: we will be setting inline-style-width/height on this element
      ref={containerRef}
    >
      <CodeEditor
        onSave={onChange}
        onBlur={onChange}
        monacoOptions={options}
        language="promql"
        value={initialValue}
        onBeforeEditorMount={(monaco) => {
          setupPromQL(
            monaco,
            () => lpRef.current,
            () => historyRef.current
          );
        }}
        onEditorDidMount={(editor) => {
          // this code makes the editor resize itself so that the content fits
          // (it will grow taller when necessary)
          const updateElementHeight = () => {
            const containerDiv = containerRef.current;
            if (containerDiv !== null) {
              const pixelHeight = editor.getContentHeight();
              const pixelWidth = containerDiv.clientWidth;
              containerDiv.style.height = `${pixelHeight}px`;
              editor.layout({ width: pixelWidth, height: pixelHeight });
            }
          };

          editor.onDidContentSizeChange(updateElementHeight);
          updateElementHeight();
        }}
      />
    </div>
  );
};
