
export function Mymonies_add_pattern(server_address: string, add_pattern_req: any, onSuccess: Function, onError: ErrorCallback): void;
export function Mymonies_list_accounts(server_address: string, list_accounts_req: any, onSuccess: Function, onError: ErrorCallback): void;
export function Mymonies_list_tags(server_address: string, list_tags_req: any, onSuccess: (res: ListTagsResponse) => void, onError: ErrorCallback): void;
export function Mymonies_list_transactions(server_address: string, list_transactions_req: ListTransactionRequest, onSuccess: (res: ListTransactionResponse) => void, onError: ErrorCallback): void;
export function Mymonies_update_tag(server_address: string, update_tag_req: any, onSuccess: Function, onError: ErrorCallback): void;

/*~ You can declare types that are available via importing the module */
interface ErrorCallback {
    (e: twirpError): void;
}

interface twirpError {
    code: string;
    msg: string;
    meta: {
        cause: string;
    }
}

export interface ListTransactionRequest {
    filter: TransactionFilter;
}

interface ListTransactionResponse {
    transactions: Transaction[];
}

interface Transaction {
    id: string;
    transaction_date: string;
    value_date: string;
    payment_date: string;
    amount: number;
    payee_payer: string;
    account: string;
    bic: string;
    transaction: string;
    reference: string;
    payer_reference: string;
    message: string;
    card_number: string;
    tag_id: string;
    import_id: string;
}

export interface TransactionFilter {
    id?: string;
    account?: string;
    month?: string;
    query?: string;
}

export interface ListTagsResponse {
    tags: Tag[];
}

export interface Tag {
    id: string;
    name: string;
}

/*~ You can declare properties of the module using const, let, or var */
export const myField: number;

/*~ If there are types, properties, or methods inside dotted names
 *~ of the module, declare them inside a 'namespace'.
 */
export namespace subProp {
    /*~ For example, given this definition, someone could write:
     *~   import { subProp } from 'yourModule';
     *~   subProp.foo();
     *~ or
     *~   import * as yourMod from 'yourModule';
     *~   yourMod.subProp.foo();
     */
    export function foo(): void;
}
