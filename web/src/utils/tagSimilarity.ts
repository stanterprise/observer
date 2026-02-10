/**
 * Calculate Jaccard similarity between two sets
 * Jaccard(A, B) = |A ∩ B| / |A ∪ B|
 */
export function jaccardSimilarity(setA: Set<string>, setB: Set<string>): number {
  if (setA.size === 0 && setB.size === 0) return 1;
  if (setA.size === 0 || setB.size === 0) return 0;

  const intersection = new Set([...setA].filter((x) => setB.has(x)));
  const union = new Set([...setA, ...setB]);

  return intersection.size / union.size;
}

/**
 * Calculate tag co-occurrence similarity based on shared tests
 * Returns a value between 0 (never co-occur) and 1 (always co-occur)
 */
export function tagCoOccurrence(
  tag1Tests: Set<string>,
  tag2Tests: Set<string>,
): number {
  return jaccardSimilarity(tag1Tests, tag2Tests);
}

/**
 * Build a similarity matrix for all tags
 * @returns Map<tag1_tag2, similarity>
 */
export function buildTagSimilarityMatrix(
  tags: Map<string, Set<string>>,
): Map<string, number> {
  const matrix = new Map<string, number>();
  const tagNames = Array.from(tags.keys());

  for (let i = 0; i < tagNames.length; i++) {
    for (let j = i + 1; j < tagNames.length; j++) {
      const tag1 = tagNames[i];
      const tag2 = tagNames[j];
      const tests1 = tags.get(tag1)!;
      const tests2 = tags.get(tag2)!;

      const similarity = tagCoOccurrence(tests1, tests2);
      const key = [tag1, tag2].sort().join("_");
      matrix.set(key, similarity);
    }
  }

  return matrix;
}

/**
 * Get similarity between two tags from the matrix
 */
export function getTagSimilarity(
  matrix: Map<string, number>,
  tag1: string,
  tag2: string,
): number {
  if (tag1 === tag2) return 1;
  const key = [tag1, tag2].sort().join("_");
  return matrix.get(key) ?? 0;
}
