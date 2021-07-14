import { HistoryItem } from '@grafana/data';
import { promLanguageDefinition } from 'monaco-promql';
import { PromQuery } from '../../types';
import type * as monacoType from 'monaco-editor/esm/vs/editor/editor.api';
type Monaco = typeof monacoType;
import { getSuggestion } from './suggestions';
import { getCompletions } from './completions';
import type PromQlLanguageProvider from '../../language_provider';

type PromHistoryItem = HistoryItem<PromQuery>;

function getMonacoCompletionItemProvider(
  monaco: Monaco,
  getLP: () => PromQlLanguageProvider,
  getHistory: () => PromHistoryItem[]
): monacoType.languages.CompletionItemProvider {
  const provideCompletionItems = (
    model: monacoType.editor.ITextModel,
    position: monacoType.Position
  ): monacoType.languages.ProviderResult<monacoType.languages.CompletionList> => {
    console.log('pci');
    const word = model.getWordAtPosition(position);
    const range =
      word != null
        ? monaco.Range.lift({
            startLineNumber: position.lineNumber,
            endLineNumber: position.lineNumber,
            startColumn: word.startColumn,
            endColumn: word.endColumn,
          })
        : monaco.Range.fromPositions(position);
    // documentation says `position` will be "adjusted" in `getOffsetAt`
    // i don't know what that means, to be sure i clone it
    const positionClone = {
      column: position.column,
      lineNumber: position.lineNumber,
    };
    const offset = model.getOffsetAt(positionClone);
    const suggestion = getSuggestion(model.getValue(), offset);
    const completionsPromise =
      suggestion != null ? getCompletions(suggestion, getLP(), getHistory()) : Promise.resolve([]);
    return completionsPromise.then((items) => {
      const suggestions = items.map((item) => ({
        kind: monaco.languages.CompletionItemKind.Text,
        label: item.label,
        insertText: item.insertText,
        range,
        command: item.suggestOnInsert
          ? {
              id: 'editor.action.triggerSuggest',
              title: '',
            }
          : undefined,
      }));
      return { suggestions };
    });
  };

  return {
    triggerCharacters: ['{', ',', '[', '('],
    provideCompletionItems,
  };
}

export const setupPromQL = (
  monaco: Monaco,
  getLP: () => PromQlLanguageProvider,
  getHistory: () => PromHistoryItem[]
) => {
  const langId = promLanguageDefinition.id;
  monaco.languages.register(promLanguageDefinition);
  promLanguageDefinition.loader().then((mod) => {
    monaco.languages.setMonarchTokensProvider(langId, mod.language);
    monaco.languages.setLanguageConfiguration(langId, mod.languageConfiguration);
    const completionProvider = getMonacoCompletionItemProvider(monaco, getLP, getHistory);
    monaco.languages.registerCompletionItemProvider(langId, completionProvider);
  });

  // FIXME: should we unregister this at end end?
};
