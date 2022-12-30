 // https://jsfiddle.net/bc_rikko/gbpw2q9x/3/

 function isElementInViewport(el) {
     var rect = el.getBoundingClientRect();

     return rect.bottom > 0 &&
         rect.right > 0 &&
         rect.left < (window.innerWidth || document.documentElement.clientWidth) /* or $(window).width() */ &&
         rect.top < (window.innerHeight || document.documentElement.clientHeight) /* or $(window).height() */ ;
 }


 function handler() {
     const el2 = document.getElementById('loadmore')
     console.log("calliong hnadler 2")


     if (typeof (el2) != 'undefined' && el2 != null) {

         console.log("calliong hnadler " + el2 + " " + isElementInViewport(el2))

         if (isElementInViewport(el2)) {

             el2.click();

         }
     }
 }

 $(document).ready(function () {
     console.log("ready! autoload");
     document.getElementById("resulttable2").addEventListener("scroll", handler);



 });