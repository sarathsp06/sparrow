import type { APIRoute } from 'astro';
import { generateLlmsTxt } from './generator';
import { llmsFullDescription } from '../prompts.mjs';

export const prerender = true;

export const GET: APIRoute = async (context) => {
  const body = await generateLlmsTxt(context, {
    minify: false,
    description: llmsFullDescription,
  });
  return new Response(body);
};
