const { Client } = require('pg')
const { validator:{validateAll} } = require('indicative') 
const { getRun } = require('util-header')

module.exports.handler = async (event, context) => {
    
    context.callbackWaitsForEmptyEventLoop = false
    let run = null
    try {
        run = await getRun(event) 
        
    } catch (error) {
        console.error('Error al obtener el run del header:: ', error)
        return { statusCode: 200 , ok : false }
    }

    const client = new Client() 

    
    try {
        await validateAll( event.body, 
            { 
                'circun': 'required|number',
                'mesa': 'required|number'
            }, 
            {
                'required':  ' El campo {{ field }} es obligatorio',
                'number':  'El campo {{ field }} no es un numero'
            })


        await client.connect()

        const { rows } =  await client.query(
            ` 
                SELECT  
                *
                FROM  mesa
                join candidato on  concat('CIRCUNSCRIPCIÓN SENATORIAL ', mesa.cs )= candidato.territorio 
                where 
                mesa.circunscripcionid=$1
                and mesa.mesa = $2
                 and (mesa.estado !=1 or mesa.estado is null) 
            `, [ event.body.circun, event.body.mesa ])  

        await client.end()

        console.log(run, event.body.circun, event.body.mesa )

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