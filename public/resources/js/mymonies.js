window.onload = init;

let app;

function init() {
    app = new Vue({
        el: '#app',
        data: {
            modalTransaction: false,
            account: '',
            accounts: [],
            tags: {},
            transactions: [],
        },
        watch: {
            account: function () {
                updateTransactions(app.account).catch(onXhrFail);
            }
        }
    });

    // Close modal on escape key. This can't be bound in div so Vue can't handle it.
    document.addEventListener("keyup", function onKeyUp (e) {
        if (e.key === "Escape") {
            app.modalTransaction = false
        }
    });


    Mymonies_list_tags("", {}, gotTags, onXhrFail);
    function gotTags(res) {
        console.log(res.tags);
        res.tags.forEach((t) => app.tags[t.name] = t.id);
        console.log(app.tags);
    }

    Mymonies_list_accounts("", {}, gotAccounts, onXhrFail);
    function gotAccounts(res) {
        app.accounts = res.accounts;
    }
}


Vue.component('tabs', {
    template: `
        <div id="tabs" class="container-fluid">
            <nav class="navbar navbar-expand navbar-light bg-light">
                <a class="navbar-brand" href="#" @click="selectTab('')">Mymonies</a>

                <ul class="navbar-nav mr-auto">
                    <li class="nav-item active" v-for="tab in tabs" v-if="tab.id">
                        <a class="nav-link" :href="tab.href" @click="selectTab(tab.id)">{{ tab.name }}</a>
                    </li>
                </ul>
            </nav>
            <div id="tabs-details">
                <slot></slot>
            </div>
        </div>
        `,
    methods: {
        selectTab(id) {
            this.tabs.forEach((tab) => {
                tab.isActive = (tab.id === id);
            });
        }
    },
    data() {
        return { tabs: [] };
    },
    created() {
        this.tabs = this.$children;
    },
    mounted() {
        this.$children.forEach((e) => {
            if (document.location.hash === e.href
                || ( document.location.hash === '' && e.href === '#')) {
                e.isActive = true;
            }
        });
    },
});

Vue.component('tab', {
    template: `
        <div v-show="isActive">
            <h1>{{ name }}</h1>
            <slot></slot>
        </div>
    `,
    props: {
        id: { required: true },
        name: { required: true },
    },
    data() {
        return { isActive: false };
    },
    computed: {
        href() {
            return '#' + this.id;
        }
    },
});

Vue.component('transaction-list', {
    template: `
        <div>
            <slot></slot>
        </div>`,
});

Vue.component('select-account', {
    template: `
        <select v-model="selected">
            <option>--</option>
            <option v-for="a in accounts">{{ a.number }}</option>
        </select>
    `,
    watch: {
        selected () { this.$emit('update:selected', this.selected) }
    },
    data() {
        return {
            selected: "--"
        };
    },
    props: {
        accounts: { required: true }
    }
});

Vue.component('tag', {
    template: `
        <select v-model="selected">
                <option v-bind:value="tags[tag]" v-for="(id, tag) in tags">{{ tag }}</option>
        </select>`,
    props: {
        selected: {},
        tags: { required: true },
    }
});

Vue.component('modal', {
    template: `
        <transition name="modal">
            <div class="modal-mask" @click="$emit('close')">
                <div class="modal-wrapper">
                    <div class="modal-container" @click.stop>

                    <div class="modal-header">
                        <slot name="header">
                            Tapahtuman tiedot
                        </slot>
                    </div>

                    <div class="modal-body">
                        <slot name="body">
                            default body
                        </slot>
                    </div>

                    <div class="modal-footer">
                        <slot name="footer">
                        <button class="modal-default-button" @click="$emit('close')">
                            OK
                        </button>
                        </slot>
                    </div>

                    </div>
                </div>
                </div>
        </transition>`
});

Vue.component('transactiondetail', {
    template: '#transactiondetail-template'
});


/*
* Replace the whole page with text "failed to load".
*/
function onXhrFail(err) {
    let body = document.getElementsByTagName("body")[0];
    body.innerHTML = '<h1>Failed to load data: ' + err + '</h1>';
}

async function updateTransactions(account) {
    Mymonies_list_transactions("", {
        filter: {
            account: account
        },
    }, (data) => app.transactions = data.transactions, onXhrFail);
}

