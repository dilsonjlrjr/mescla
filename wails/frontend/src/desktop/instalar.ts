// Importado primeiro em main.ts: as trocas precisam valer antes de qualquer
// módulo da interface rodar código de topo que chame a API ou leia receitas.
import { instalarPonteApi } from './ponteApi';
import { instalarDownloadNativo } from './downloadNativo';
import { migrarReceitasAntigas } from './migrarReceitas';

instalarPonteApi();
instalarDownloadNativo();
migrarReceitasAntigas();
