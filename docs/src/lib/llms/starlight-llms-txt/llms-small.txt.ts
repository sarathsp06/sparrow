import type { APIRoute } from 'astro';
import { generateLlmsTxt } from './generator';
import { llmsSmallDescription } from '../prompts.mjs';
import { starlightLllmsTxtContext } from 'virtual:starlight-llms-txt/context';

export const prerender = true;

export const GET: APIRoute = async (context) => {
  const body = await generateLlmsTxt(context, {
    minify: true,
    description: llmsSmallDescription,
    exclude: starlightLllmsTxtContext.exclude,
  });
  return new Response(body);
};
