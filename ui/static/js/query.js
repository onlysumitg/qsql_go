 let sqlrunner_vue = new Vue({

     el: "#sqlrunner",
     delimiters: ['[[', ']]'],

     data() {
         return {
             contentA: sessionStorage.getItem("adhoc_sql") || "-- Cntl+E: Execute current line or Selected text\n ---------------- \n",
             contentB: "",//sessionStorage.getItem("contentbsaved") || "",

             contentR: "",
             sqlresults: [],
             showModal: false,
             singleRowData: {},
             singleRow: [],
             singleRowColumns: [],
             singleRowIndex: 0,
             showSinglRow: false,
             showSinglColumn: false,
             singleColumnData: {},
             singleColumnValue: "",
             singleColumnMode: "text",
             processing: false,
             splitter: {},
             id: 0,
             resulttable2key:0,

             savesQueries:{},
             queryFields:[],
             currentQuery:{},


             searchText:"",
             searchCatagory:"*ALL"
          }
     },

     computed: {
        filteredCatagoryList() {
            let tempSearch = this.searchCatagory.toUpperCase();
            if (tempSearch.trim().lenthg == 0) return JSON.parse(JSON.stringify(this.savesQueries));

            if (tempSearch.trim() == "*ALL") return JSON.parse(JSON.stringify(this.savesQueries));

            let tempsavesQueries = JSON.parse(JSON.stringify(this.savesQueries))
     

            let returnobj = {}
            returnobj[tempSearch] = tempsavesQueries[tempSearch]
            return  returnobj
        },


          filteredQueryList() {



            let tempSearch = this.searchText.toUpperCase();
            if (tempSearch.trim().lenthg == 0) return this.savesQueries;

            let tempsavesQueries = JSON.parse(JSON.stringify(this.filteredCatagoryList))
             
            for (let k of Object.keys(tempsavesQueries)) {
     
              tempsavesQueries[k] = tempsavesQueries[k].filter(query => {
                  return (
                      query.name.toUpperCase().match(tempSearch) ||
                      query.category.toUpperCase().match(tempSearch) 
                    //  || query.sql.toUpperCase().match(tempSearch)  
                    );
              })
            } 
  
            return tempsavesQueries
          
          },
      },




     mounted() {
        try {
        if (splitmode ==3) {
            this.split3()
        }
        else{
            this.split2()
        } } catch (error) {
            console.error(error)
            this.split2()
         }

         try {
         this.sqlresults = JSON.parse(initialResultData)}
         catch (error) {
            //console.error(error)

         }

         try {
         this.savesQueries= JSON.parse(savesQueries)
         }
         catch (error) {
           // console.error(error)

         }

         try{
            this.contentB = contentB
         }
         catch (error) {
            // console.error(error)
 
          }
 
     },
     methods: {

   
 

        showForm(query){
            this.currentQuery = query
            this.queryFields = query.fields
        },



        split3(){
            let self = this

            var sizes = localStorage.getItem('split-three')
   
            if (sizes) {
                sizes = JSON.parse(sizes)
            } else {
                sizes = [25,25, 50] // default sizes
            }
   
   
            this.splitter = Split(['#leftone','#midone', '#rightone'], {
   
   
                sizes: sizes,
                minSize: [100,100, 300],
                onDragEnd: function (sizes) {
                    localStorage.setItem('split-three', JSON.stringify(sizes))
                    self.rerender()
                },
            })
        },
        split2(){
            let self = this

            var sizes = localStorage.getItem('split-two')
   
            if (sizes) {
                sizes = JSON.parse(sizes)
            } else {
                sizes = [25, 75] // default sizes
            }
   
   
            this.splitter = Split(['#leftone', '#rightone'], {
   
   
                sizes: sizes,
                minSize: [100, 300],
                onDragEnd: function (sizes) {
                    localStorage.setItem('split-two', JSON.stringify(sizes))
                    self.rerender()
                },
            })
        },


         rerender() {
             this.id++
         },
         previous_row() {
             let index = this.singleRowIndex - 1
             if (index < 0) {
                 index = 0
             }
             this.updateSingleRowIndex(index)

         },

         next_row() {

             let index = this.singleRowIndex + 1

             if (index >= this.singleRowData.Rows.length) {
                 index = this.singleRowData.Rows.length - 1
             }
             this.updateSingleRowIndex(index)

         },
         closeSingleRow() {
             this.showSinglRow = false
         },
         closeSingleColumn() {
             this.showSinglColumn = false
         },
         updateSingleRowIndex(index) {

             this.singleRowIndex = index
             this.showSinglRow = true
             this.singleRow = this.singleRowData.Rows[index]
             this.singleRowColumns = this.singleRowData.Columns

         },


         showSingleColumn(e, col, value) {


             this.singleColumnData = col
             this.singleColumnValue = value

             this.contentR = this.prettyPrint(value)


             this.showSinglColumn = true
             e.preventDefault();
         },

         showSingleRow(e, result, index) {

           
             this.singleRowData = result
             this.updateSingleRowIndex(index)
             e.preventDefault();
         },

         donothing(e) {
             e.preventDefault();
             e.stopPropagation();
         },
         loadMore(sqlresult) {

             let config = {
                 headers: {
                     "X-CSRF-Token": csrftoken,
                     "Content-Type": "application/json",
                     "Accept": "application/json"
                 }
             }
             let data = sqlresult.CurrentSql
             var local = this
             this.showModal = true
             local.processing = true

             axios.post('/query/loadmore', data, config).then(function (response) {

                 sqlresult.CurrentSql = response.data.CurrentSql
                 if (response.data.Rows != null) {
                     sqlresult.Rows = sqlresult.Rows.concat(response.data.Rows)
                 }






             }).catch(function (error) {

                 // handle error
                 // console.log(error);
             }).then(function () {
                 local.showModal = false
                 local.processing = false
                 handler()
                 // always executed
             });


         },
         changeContentA(val) {
             this.closeSingleRow()
             this.contentA = val
             sessionStorage.setItem("adhoc_sql", val);
         },
         changeContentB(val) {
            this.closeSingleRow()
            this.contentB = val
            //sessionStorage.setItem("contentbsaved", val);
        },
         getFieldValue(field){
            let returnValue =  localStorage.getItem(field.Name);

            if (!returnValue){
                returnValue = field.DefaultValue
            }
            return returnValue;

         },
         buildSQL(e){
            e.preventDefault()
       

            this.processing = true

            var formData = new FormData(e.target),
            result = {};
    
                for (var entry of formData.entries())
                {
                    result[entry[0]] = entry[1];
                    localStorage.setItem(entry[0], entry[1])
                }
                result = JSON.stringify(result)
            

                let config = {
                    headers: {
                        "X-CSRF-Token": csrftoken, // csrf_token
                        "Content-Type": "application/json",
                        "Accept": "application/json"
                    }
                }


             var local = this
             this.showModal = true
             axios.post('/savesql/build', result, config).then(function (response) {
                 console.log(response)
                 let sqltorun = new String(response.data.sqltorun)
                 if (typeof str === 'string' && str.length === 0) {
                        // show error
                 }
                 else{
                    local.runSQL(sqltorun)
                 }

             }).catch(function (error) {

                 // handle error
                 // console.log(error);
             }).then(function () {
                 local.showModal = false
                 //local.processing = false
                 handler()
                 // always executed
             });

         },


         runSQL(val) {

             this.closeSingleRow()

             this.processing = true


             let config = {
                 headers: {
                     "X-CSRF-Token": csrftoken, // csrf_token
                     "Content-Type": "application/json",
                     "Accept": "application/json"
                 }
             }
             let data = {
                 SQLToRun: val,

             }

             var local = this
             this.showModal = true
             axios.post('/query/run', data, config).then(function (response) {
                
                 local.sqlresults = response.data

             }).catch(function (error) {
                if (error.response.status === 303) {
                     location.reload()
                    //let data = JSON.parse(error.response.data)
                }
           
                console.error(error)
                 // handle error
                 // console.log(error);
             }).then(function () {
                 local.showModal = false
                 local.processing = false
                 local.resulttable2key += 1
                 //handler()
                 // always executed
             });


         },
        

         prettyPrint: function (stringval) {

             let returnString = stringval
             this.singleColumnMode = "text"

             try {
                 returnString = formatJson(stringval)
                 this.singleColumnMode = "json"
            
                 return returnString

             } catch (error) {
                //console.error(error)
             }

             try {
                 returnString = formatXML(stringval)
                 this.singleColumnMode = "xml"
            
                 return returnString
             } catch (error) {
                //console.error(error)
             }

             if (returnString != stringval) {
                 returnString = stringval
             }
             return returnString

         },

    

     }
 });

 