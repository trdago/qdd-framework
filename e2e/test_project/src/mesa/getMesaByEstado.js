const { Client } = require('pg')
const { getRun } = require('util-header')
const { validator:{validateAll} } = require('indicative') 

module.exports.handler = async (event, context) => {
    

    context.callbackWaitsForEmptyEventLoop = false

    let run = null
    try {
        run = await getRun(event) 
    } catch (error) {
        console.error('Error al obtener el run del header:: ', error)
    }

    const client = new Client() 
    
    try {

        await validateAll( event.body, 
            { 
                'estado_filter': 'number'
            }, 
            {
                'required':  ' El campo {{ field }} es obligatorio',
                'number':  'El campo {{ field }} no es un numero'
            })
        const data=event.body;
        const estadoFilter = data ? data.estado_filter != null ? data.estado_filter : null  : null

        await client.connect()

        let values=[]
        if(!estadoFilter){
            values =[
            ]}
        else{
                values=[ estadoFilter]
        }

   const { rows } =  await client.query(
    ` 
        SELECT  
        *
        FROM  mesa
        join usuario on usuario.id = mesa.usuario_id
        where 
         ${estadoFilter ? ' mesa.estado = $1' : 'mesa.estado is not null'}
         order by mesa.circunscripcionid, mesa.mesa asc
        limit 5
        
    `, values)  

        await client.end()

        return { statusCode: 200 , data : rows }


    }catch(error){
        console.error('Fallo la consulta::', error)
        if(client)
        {
            await client.end()

        }
        return { statusCode: 200 , ok : false }

    }
}