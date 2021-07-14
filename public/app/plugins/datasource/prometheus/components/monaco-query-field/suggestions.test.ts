import { getSuggestion } from './suggestions';
import type { Suggestion } from './suggestions';

// we use the `^` character as the cursor-marker in the string.
function assertSuggestion(situation: string, expectedSuggestion: Suggestion | null) {
  // first we find the cursor-position
  const pos = situation.indexOf('^');
  if (pos === -1) {
    throw new Error('cursor missing');
  }

  // we remove the cursor-marker from the string
  const text = situation.replace('^', '');

  // sanity check, make sure no more cursor-markers remain
  if (text.indexOf('^') !== -1) {
    throw new Error('multiple cursors');
  }

  const result = getSuggestion(text, pos);

  if (expectedSuggestion === null) {
    expect(result).toStrictEqual(null);
  } else {
    expect(result).toMatchObject(expectedSuggestion);
  }
}

describe('suggestions', () => {
  it('handles things', () => {
    assertSuggestion('^', {
      type: 'ALL_METRIC_AND_FUNCTION_NAMES_AND_HISTORY',
    });

    assertSuggestion('sum(one) / ^', {
      type: 'ALL_METRIC_AND_FUNCTION_NAMES',
    });

    assertSuggestion('sum(^)', {
      type: 'ALL_METRIC_NAMES',
    });

    assertSuggestion('sum(one) / sum(^)', {
      type: 'ALL_METRIC_NAMES',
    });

    assertSuggestion('something{}[^]', {
      type: 'ALL_DURATIONS',
    });

    assertSuggestion('something{^}', {
      type: 'LABEL_NAMES_FOR_SELECTOR',
      metricName: 'something',
      otherLabels: [],
    });

    assertSuggestion('sum(something) by (^)', {
      type: 'LABEL_NAMES_FOR_BY',
      metricName: 'something',
      otherLabels: [],
    });

    assertSuggestion('sum by (^) (something)', {
      type: 'LABEL_NAMES_FOR_BY',
      metricName: 'something',
      otherLabels: [],
    });

    assertSuggestion('something{one="val1",two="val2",^}', {
      type: 'LABEL_NAMES_FOR_SELECTOR',
      metricName: 'something',
      otherLabels: [
        { name: 'one', value: 'val1' },
        { name: 'two', value: 'val2' },
      ],
    });

    assertSuggestion('something{job=^}', {
      type: 'LABEL_VALUES',
      metricName: 'something',
      labelName: 'job',
      otherLabels: [],
    });

    assertSuggestion('something{job=^,host="h1"}', {
      type: 'LABEL_VALUES',
      metricName: 'something',
      labelName: 'job',
      otherLabels: [{ name: 'host', value: 'h1' }],
    });

    assertSuggestion('something{one="val1",two="val2",three=^,four="val4",five="val5"}', {
      type: 'LABEL_VALUES',
      metricName: 'something',
      labelName: 'three',
      otherLabels: [
        { name: 'one', value: 'val1' },
        { name: 'two', value: 'val2' },
        { name: 'four', value: 'val4' },
        { name: 'five', value: 'val5' },
      ],
    });
  });
});
