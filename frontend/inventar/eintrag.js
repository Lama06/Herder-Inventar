import ProblemEintrag from "./problem_eintrag.js"

export default {
    props: ["obj"],
    emits: ["loeschen", "aendern"],
    components: {
        ProblemEintrag
    },
    data() {
        return {
            nameEingabe: this.obj.name,
            raumEingabe: this.obj.raum,

            problemeAnzeigen: false,
        }
    },
    computed: {
        aenderungenGemacht() {
            return this.nameEingabe !== this.obj.name || this.raumEingabe !== this.obj.raum
        }
    },
    methods: {
        problemLoesen(index) {
            let kopie = JSON.parse(JSON.stringify(this.obj.probleme))
            let neu = kopie.filter((_, i) => i !== index)
            this.$emit("aendern", {
                ...this.obj,
                probleme: neu
            })
        },
        async loeschen() {
            let response;
            try {
                response = await fetch("/api/objekte/loeschen/", {
                    method: "POST",
                    body: JSON.stringify({
                        schlüssel: sessionStorage.getItem("schlüssel"),
                        id: this.obj.id
                    })
                })
            } catch {
                alert("Server nicht erreichbar");
                return
            }

            if (!response.ok) {
                alert("Fehler")
                return
            }

            this.$emit("loeschen")
        },
        async aenderungenSpeichern() {
            let response;
            try {
                response = await fetch("/api/objekte/aendern/", {
                    method: "POST",
                    body: JSON.stringify({
                        schlüssel: sessionStorage.getItem("schlüssel"),
                        id: this.obj.id,
                        name: this.nameEingabe,
                        raum: this.raumEingabe,
                    })
                })
            } catch {
                alert("Server nicht erreichbar");
                return
            }

            if (!response.ok) {
                alert("Fehler");
                return
            }

            this.$emit("aendern", {
                ...this.obj,
                name: this.nameEingabe,
                raum: this.raumEingabe,
            })
        },
    },
    template: `
<div>
    <h3 :title="obj.id">{{ obj.name }}</h3>
    <span>Name: </span> <input type="text" v-model="nameEingabe">
    <br>

    Raum: <input type="text" v-model="raumEingabe">
    <br>
    
    <template v-if="obj.probleme.length != 0">
        Es gibt {{ obj.probleme.length }} Probleme! 
        <button @click="problemeAnzeigen = !problemeAnzeigen">{{ problemeAnzeigen ? "Ausblenden" : "Anzeigen" }}</button>
        <br>
    </template>
    
    <template v-if="problemeAnzeigen">
        <ProblemEintrag
            v-for="(problem, index) of obj.probleme"
            :key="problem.id"
            :problem="problem"
            :obj="obj"
            :index="index"
            @loesen="problemLoesen(index)"
        >
        </ProblemEintrag>
    </template>
    
    <button @click="loeschen">Löschen</button>
    <button v-if="aenderungenGemacht" @click="aenderungenSpeichern">Änderungen speichern</button>
    <hr>
</div>
`
}
