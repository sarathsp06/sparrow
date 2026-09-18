import type { APIRoute, GetStaticPaths, InferGetStaticParamsType, InferGetStaticPropsType } from 'astro';
import { starlightLllmsTxtContext } from 'virtual:starlight-llms-txt/context';
import { generateLlmsTxt } from './generator';

export const prerender = true;

export const getStaticPaths = (() => {
  return starlightLllmsTxtContext.customSets.map(({ label, description, paths, slug }) => ({
    params: { slug },
    props: { label, description, paths },
  }));
}) satisfies GetStaticPaths;

type Props = InferGetStaticPropsType<typeof getStaticPaths>;
type Params = InferGetStaticParamsType<typeof getStaticPaths>;

export const GET: APIRoute<Props, Params> = async (context) => {
  let description = context.props.label;
  if (context.props.description) description += ': ' + context.props.description;
  const body = await generateLlmsTxt(context, {
    minify: true,
    description,
    include: context.props.paths,
  });
  return new Response(body);
};
