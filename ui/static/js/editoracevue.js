 

 Vue.component('Editor', {
     template: '<div :id="editorId" class="maxheight72 my-sticky-top" style=" "></div>',
     props: ['editorId', 'content', 'lang', 'theme', 'setreadonly'],
     data() {
         return {
             editor: Object,
             beforeContent: ''
         }
     },
     watch: {
         'content'(value) {
             if (this.beforeContent !== value) {
                 this.editor.setValue(new String(value), 1)
                
             }
         }
     },


     mounted() {
         const lang = this.lang || 'sql'
         const theme = this.theme || 'chrome'

         this.editor = ace.edit(this.editorId)
 

         this.editor.setValue(new String(this.content), 1)
         this.editor.setOption("selectionStyle", "line");
         this.editor.setOption("behavioursEnabled", true);
         this.editor.setOption("wrapBehavioursEnabled", true);
         this.editor.setOption("autoScrollEditorIntoView", true);

         this.editor.setOption("fontFamily", "Roboto Mono");
         this.editor.setOption("fontSize", "11pt");


         this.editor.setOption("wrap", true);
         // this.editor.setOption("showGutter", false);
         this.editor.setOption("showPrintMargin", false)

         this.editor.getSession().setMode(`ace/mode/${lang}`)
         this.editor.setTheme(`ace/theme/${theme}`)


 
         if (this.setreadonly) {
             this.editor.setReadOnly(true)
         } else {
             this.editor.setReadOnly(false)
         }

         this.editor.on('change', () => {
             this.beforeContent = this.editor.getValue()
             this.$emit('change-content', this.editor.getValue())
         })

         this.editor.focus()
         let component = this
         this.editor.commands.addCommand({
             name: 'myCommand',
             bindKey: {
                 win: 'Ctrl-E',
                 mac: 'Command-E'
             },
             exec: function (editor) {
                 var currline = editor.getSelectionRange().start.row;
                 var current_line_text = editor.session.getLine(currline);

                 sql_to_run = editor.getSelectedText() || current_line_text
                 component.$emit('run-sql', sql_to_run)




             },
             readOnly: false, // false if this command should not apply in readOnly mode
             // multiSelectAction: "forEach", optional way to control behavior with multiple cursors
             // scrollIntoView: "cursor", control how cursor is scolled into view after the command
         });




     }
 })




 