 // https://jsfiddle.net/bc_rikko/gbpw2q9x/3/

 function isElementInViewport(el) {
     var rect = el.getBoundingClientRect();

     return rect.bottom > 0 &&
         rect.right > 0 &&
         rect.left < (window.innerWidth || document.documentElement.clientWidth) /* or $(window).width() */ &&
         rect.top < (window.innerHeight || document.documentElement.clientHeight) /* or $(window).height() */ ;
 }


 function handler() {
    let array1 = document.getElementsByClassName('loadmore')
    //  const el2 = document.getElementById('loadmore')
    //  console.log("calliong hnadler 2")
    for (const el2 of array1) {
        if (isElementInViewport(el2)) { el2.click();}
    }

   
    
 }

 