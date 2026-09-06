// Carregamento e transições — o indicador de espera é a própria logomarca
// girando, e a troca de tela é uma transição, não um corte seco.
//
// TDD unit-only: o Spinner é verificado no DOM; as regras que vivem só em CSS
// (a tela que sai não pode usar `display: none`, e todo movimento morre sob
// prefers-reduced-motion) são verificadas no código-fonte, porque jsdom não
// resolve cascata de folha externa nem media query de movimento.

import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';
import { render } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';
import Spinner from '../src/lib/components/Spinner.svelte';

const here = dirname(fileURLToPath(import.meta.url));
const src = (rel: string) => readFileSync(join(here, '..', rel), 'utf8');

describe('Spinner — logomarca girando', () => {
  it('anuncia o carregamento a leitor de tela', () => {
    const { getByRole } = render(Spinner, { label: 'Carregando o catálogo' });
    const status = getByRole('status');
    expect(status).toHaveAttribute('aria-label', 'Carregando o catálogo');
  });

  it('desenha a marca, não um spinner genérico', () => {
    const { getByRole, container } = render(Spinner);
    // A marca é o d20 de 7 facetas do protótipo.
    expect(container.querySelectorAll('svg polygon')).toHaveLength(7);
    expect(getByRole('status')).toBeInTheDocument();
  });

  it('respeita o tamanho pedido', () => {
    const { getByRole } = render(Spinner, { size: 40 });
    expect(getByRole('status').getAttribute('style')).toContain('--spin-size: 40px');
  });

  it('gira num tempo configurável', () => {
    const { getByRole } = render(Spinner, { duration: 1200 });
    expect(getByRole('status').getAttribute('style')).toContain('--spin-duration: 1200ms');
  });
});

describe('Transição entre telas', () => {
  const app = src('src/App.svelte');

  it('a tela que sai some por opacidade, não por display: none', () => {
    // `display` não é animável — voltar a usá-lo aqui derruba a transição
    // inteira sem quebrar mais nada, então a regra fica travada em teste.
    const regra = /\.view\.hidden\s*\{[^}]*\}/.exec(app);
    expect(regra, 'App.svelte sem a regra .view.hidden').not.toBeNull();
    expect(regra![0]).not.toMatch(/display:\s*none/);
    expect(regra![0]).toMatch(/opacity:\s*0/);
    expect(regra![0]).toMatch(/visibility:\s*hidden/);
    expect(regra![0]).toMatch(/pointer-events:\s*none/);
  });

  it('a tela escondida não recebe foco nem toque', () => {
    expect(app).toMatch(/\.view\s*\{[^}]*transition:/);
  });
});

describe('Movimento reduzido', () => {
  it('todo o movimento morre sob prefers-reduced-motion', () => {
    const css = src('src/app.css');
    expect(css).toMatch(/@media\s*\(prefers-reduced-motion:\s*reduce\)/);
    expect(css).toMatch(/animation-duration:\s*0\.001ms\s*!important/);
    expect(css).toMatch(/transition-duration:\s*0\.001ms\s*!important/);
  });
});
