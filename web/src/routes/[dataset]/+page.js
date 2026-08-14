import { error } from "@sveltejs/kit";
import { DATASETS, datasetById } from "$lib/datasets";

/**
 * One prerendered page per registered dataset. The registry is the repository
 * root's datasets.json, so a dataset the assembler builds is a page that exists,
 * and one it does not is a 404 at build time rather than a blank page.
 */
export const entries = () => DATASETS.map((d) => ({ dataset: d.id }));

export const load = ({ params }) => {
  const dataset = datasetById(params.dataset);
  if (!dataset) throw error(404, `Unknown dataset: ${params.dataset}`);
  return { dataset };
};
