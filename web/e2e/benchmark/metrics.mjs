export function round(value, digits = 1) {
  const scale = 10 ** digits;
  return Math.round(value * scale) / scale;
}

export function percentile(values, percentileValue) {
  if (values.length === 0) return 0;

  const sorted = [...values].sort((a, b) => a - b);
  const index = Math.min(
    sorted.length - 1,
    Math.max(0, Math.ceil((percentileValue / 100) * sorted.length) - 1),
  );

  return round(sorted[index]);
}

export function average(values) {
  if (values.length === 0) return 0;

  return round(values.reduce((sum, value) => sum + value, 0) / values.length);
}

export function summarizeStep(step, durations, pass, fail, elapsedSeconds) {
  return {
    step,
    type: 'Real',
    avg: average(durations),
    p50: percentile(durations, 50),
    p95: percentile(durations, 95),
    p99: percentile(durations, 99),
    pass,
    fail,
    throughput: round(pass / Math.max(elapsedSeconds, 0.001), 2),
  };
}

export function buildBenchmarkSummary(totalDurations, passedIterations, totalIterations, elapsedSeconds) {
  return {
    avg: average(totalDurations),
    p50: percentile(totalDurations, 50),
    p95: percentile(totalDurations, 95),
    p99: percentile(totalDurations, 99),
    successRate: round((passedIterations / Math.max(totalIterations, 1)) * 100),
    passedIterations,
    totalIterations,
    submittedThroughput: round(totalIterations / Math.max(elapsedSeconds, 0.001), 2),
    completedThroughput: round(passedIterations / Math.max(elapsedSeconds, 0.001), 2),
  };
}

export function formatDuration(value) {
  return `${value.toFixed(1)}ms`;
}

export function formatThroughput(value) {
  return `${value.toFixed(2)}/s`;
}

function pad(value, width, align = 'left') {
  const text = String(value);
  if (text.length >= width) return text;

  const spaces = ' '.repeat(width - text.length);
  return align === 'right' ? `${spaces}${text}` : `${text}${spaces}`;
}

export function formatStepTable(summaries) {
  const rows = summaries.map((summary) => ({
    Step: summary.step,
    Type: summary.type,
    Avg: formatDuration(summary.avg),
    P50: formatDuration(summary.p50),
    P95: formatDuration(summary.p95),
    P99: formatDuration(summary.p99),
    Pass: String(summary.pass),
    Fail: String(summary.fail),
    Throughput: formatThroughput(summary.throughput),
  }));

  const columns = ['Step', 'Type', 'Avg', 'P50', 'P95', 'P99', 'Pass', 'Fail', 'Throughput'];
  const widths = Object.fromEntries(
    columns.map((column) => [
      column,
      Math.max(column.length, ...rows.map((row) => row[column].length)),
    ]),
  );

  const formatRow = (row) =>
    `| ${columns
      .map((column) => {
        const align = ['Pass', 'Fail'].includes(column) ? 'right' : 'left';
        return pad(row[column], widths[column], align);
      })
      .join(' | ')} |`;

  const header = formatRow(Object.fromEntries(columns.map((column) => [column, column])));
  const separator = `|${columns.map((column) => '-'.repeat(widths[column] + 2)).join('|')}|`;

  return [header, separator, ...rows.map(formatRow)].join('\n');
}
