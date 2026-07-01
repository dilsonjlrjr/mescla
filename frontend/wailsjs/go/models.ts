export namespace ai {
	
	export class Response {
	
	
	    static createFrom(source: any = {}) {
	        return new Response(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}

}

export namespace main {
	
	export class ManufacturerDTO {
	    id: number;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new ManufacturerDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	    }
	}
	export class PaintDTO {
	    id: number;
	    name: string;
	    code: string;
	    manufacturer: string;
	    productLine: string;
	    r: number;
	    g: number;
	    b: number;
	    swatchPath: string;
	    thumbnail: string;
	    imageUrl: string;
	    finishType: string;
	    paintType: string;
	    coverage: string;
	    opacity: string;
	    volume: string;
	
	    static createFrom(source: any = {}) {
	        return new PaintDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.code = source["code"];
	        this.manufacturer = source["manufacturer"];
	        this.productLine = source["productLine"];
	        this.r = source["r"];
	        this.g = source["g"];
	        this.b = source["b"];
	        this.swatchPath = source["swatchPath"];
	        this.thumbnail = source["thumbnail"];
	        this.imageUrl = source["imageUrl"];
	        this.finishType = source["finishType"];
	        this.paintType = source["paintType"];
	        this.coverage = source["coverage"];
	        this.opacity = source["opacity"];
	        this.volume = source["volume"];
	    }
	}
	export class RecipeIngredientDTO {
	    paintId: number;
	    name: string;
	    percentage: number;
	    r: number;
	    g: number;
	    b: number;
	
	    static createFrom(source: any = {}) {
	        return new RecipeIngredientDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.paintId = source["paintId"];
	        this.name = source["name"];
	        this.percentage = source["percentage"];
	        this.r = source["r"];
	        this.g = source["g"];
	        this.b = source["b"];
	    }
	}
	export class RecipeDTO {
	    ingredients: RecipeIngredientDTO[];
	    resultR: number;
	    resultG: number;
	    resultB: number;
	    deltaE: number;
	    method: string;
	
	    static createFrom(source: any = {}) {
	        return new RecipeDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ingredients = this.convertValues(source["ingredients"], RecipeIngredientDTO);
	        this.resultR = source["resultR"];
	        this.resultG = source["resultG"];
	        this.resultB = source["resultB"];
	        this.deltaE = source["deltaE"];
	        this.method = source["method"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class SearchResultDTO {
	    paintId: number;
	    name: string;
	    manufacturer: string;
	    r: number;
	    g: number;
	    b: number;
	    deltaE: number;
	    swatchPath: string;
	    similarity: number;
	
	    static createFrom(source: any = {}) {
	        return new SearchResultDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.paintId = source["paintId"];
	        this.name = source["name"];
	        this.manufacturer = source["manufacturer"];
	        this.r = source["r"];
	        this.g = source["g"];
	        this.b = source["b"];
	        this.deltaE = source["deltaE"];
	        this.swatchPath = source["swatchPath"];
	        this.similarity = source["similarity"];
	    }
	}
	export class StatsDTO {
	    manufacturers: number;
	    productLines: number;
	    paints: number;
	    equivalences: number;
	    recipes: number;
	
	    static createFrom(source: any = {}) {
	        return new StatsDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.manufacturers = source["manufacturers"];
	        this.productLines = source["productLines"];
	        this.paints = source["paints"];
	        this.equivalences = source["equivalences"];
	        this.recipes = source["recipes"];
	    }
	}

}

